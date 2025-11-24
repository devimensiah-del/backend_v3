package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSupabaseStorageClient_UploadSignedURL(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/storage/v1/object/reports-pdf/"):
			w.WriteHeader(http.StatusOK)
		case strings.HasPrefix(r.URL.Path, "/storage/v1/object/sign/reports-pdf/"):
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]string{"signedURL": "/storage/v1/object/sign/reports-pdf/report.pdf?token=testtoken"}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewSupabaseStorageClient(server.URL, "reports-pdf", "token", 3600)

	url, err := client.Upload(ctx, "report.pdf", []byte("pdf"), "application/pdf")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedPrefix := server.URL + "/storage/v1/object/sign/reports-pdf/report.pdf?token=testtoken"
	if url != expectedPrefix {
		t.Fatalf("expected signed url %s, got %s", expectedPrefix, url)
	}
}
