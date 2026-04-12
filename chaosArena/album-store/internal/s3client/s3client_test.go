package s3client_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/vinaldsouza/album-store/internal/s3client"
)

// TestUploadAndDelete requires real AWS credentials and a real S3 bucket.
// Set S3_BUCKET and AWS_REGION env vars before running.
// Skip with: go test ./internal/s3client/... (will auto-skip if env vars missing)
func TestUploadAndDelete(t *testing.T) {
	bucket := os.Getenv("S3_BUCKET")
	region := os.Getenv("AWS_REGION")
	if bucket == "" || region == "" {
		t.Skip("S3_BUCKET and AWS_REGION not set, skipping integration test")
	}

	ctx := context.Background()
	client, err := s3client.New(ctx, region, bucket)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	key := fmt.Sprintf("test/smoke-%d.txt", time.Now().UnixNano())
	data := []byte("chaos arena smoke test")

	// Upload
	url, err := client.Upload(ctx, key, bytes.NewReader(data), "text/plain")
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	t.Logf("uploaded to: %s", url)

	// Verify public URL is accessible
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("fetch url: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Delete
	if err := client.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Verify gone
	resp2, err := http.Get(url)
	if err != nil {
		t.Fatalf("fetch after delete: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode == http.StatusOK {
		t.Errorf("expected non-200 after delete, got %d", resp2.StatusCode)
	}
	t.Logf("delete verified: status %d", resp2.StatusCode)
}
