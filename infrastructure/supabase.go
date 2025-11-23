package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
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
	ProjectURL string // https://xyz.supabase.co
	Bucket     string // "reports"
	Token      string // Service Role Key (for uploads)
	HTTPClient *http.Client
}

func NewSupabaseStorageClient(projectURL, bucket, token string) *SupabaseStorageClient {
	return &SupabaseStorageClient{
		ProjectURL: projectURL,
		Bucket:     bucket,
		Token:      token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *SupabaseStorageClient) Upload(ctx context.Context, path string, data []byte, contentType string) (string, error) {
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

	// Return the Public URL
	// Format: https://xyz.supabase.co/storage/v1/object/public/{bucket}/{path}
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.ProjectURL, s.Bucket, path)
	return publicURL, nil
}
