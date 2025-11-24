package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

// SupabaseStorageClient uploads files to Supabase Storage
//
// CRITICAL REQUIREMENT: The target bucket MUST be set to "Public" in Supabase Dashboard
// - Navigate to: Storage > [bucket_name] > Settings > Public bucket = ON
// - If the bucket is private, the returned public URLs will fail with 403 Forbidden
//
// Alternative for Private Buckets:
// - Use Supabase's createSignedUrl() API to generate time-limited URLs
// - Requires implementing /storage/v1/object/sign/{bucket}/{path} endpoint
type SupabaseStorageClient struct {
	ProjectURL     string // https://xyz.supabase.co
	Bucket         string // "reports"
	Token          string // Service Role Key (for uploads)
	SignedTTL      int    // seconds
	HTTPClient     *http.Client
	CircuitBreaker *gobreaker.CircuitBreaker
}

func NewSupabaseStorageClient(projectURL, bucket, token string, signedTTL int) *SupabaseStorageClient {
	// SECURITY/RELIABILITY: Add circuit breaker to prevent cascade failures
	cbSettings := gobreaker.Settings{
		Name:        "supabase-storage",
		MaxRequests: 2,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 3
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			fmt.Printf("Circuit breaker '%s' changed from %s to %s\n", name, from, to)
		},
	}

	return &SupabaseStorageClient{
		ProjectURL: projectURL,
		Bucket:     bucket,
		Token:      token,
		SignedTTL:  signedTTL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		CircuitBreaker: gobreaker.NewCircuitBreaker(cbSettings),
	}
}

func (s *SupabaseStorageClient) Upload(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	// Execute through circuit breaker
	result, err := s.CircuitBreaker.Execute(func() (interface{}, error) {
		return s.upload(ctx, path, data, contentType)
	})

	if err != nil {
		return "", fmt.Errorf("circuit breaker error: %w", err)
	}

	return result.(string), nil
}

func (s *SupabaseStorageClient) upload(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	// API Endpoint: POST /storage/v1/object/{bucket}/{path}
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.ProjectURL, s.Bucket, path)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", contentType)
	// Upsert allows overwriting if we regenerate the report
	req.Header.Set("x-upsert", "true")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase storage error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Generate signed URL (private bucket friendly)
	signURL := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s", s.ProjectURL, s.Bucket, path)
	payload := fmt.Sprintf(`{"expiresIn": %d}`, s.SignedTTL)
	signReq, err := http.NewRequestWithContext(ctx, "POST", signURL, bytes.NewBufferString(payload))
	if err != nil {
		return "", err
	}
	signReq.Header.Set("Authorization", "Bearer "+s.Token)
	signReq.Header.Set("Content-Type", "application/json")

	signResp, err := s.HTTPClient.Do(signReq)
	if err != nil {
		return "", err
	}
	defer signResp.Body.Close()

	if signResp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(signResp.Body)
		return "", fmt.Errorf("supabase sign url error (%d): %s", signResp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(signResp.Body)
	if err != nil {
		return "", err
	}

	// Supabase returns {"signedURL":"/storage/v1/object/sign/<bucket>/<path>?token=..."}
	type signRespBody struct {
		SignedURL string `json:"signedURL"`
	}
	var sr signRespBody
	if err := json.Unmarshal(bodyBytes, &sr); err != nil {
		return "", fmt.Errorf("failed to parse signed URL response: %w", err)
	}
	if sr.SignedURL == "" {
		return "", fmt.Errorf("signed URL missing in response")
	}

	// If SignedURL is relative, prefix with project URL
	if sr.SignedURL[0] == '/' {
		return s.ProjectURL + sr.SignedURL, nil
	}
	return sr.SignedURL, nil
}
