package gcs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// Client is a thin wrapper around the official GCS client with a couple of
// convenience helpers for uploading and generating signed URLs.
//
// Usage notes:
//   - The client will use Application Default Credentials when no explicit
//     credentials file is provided.
//   - If the environment variable GOOGLE_APPLICATION_CREDENTIALS is set and
//     points to a service account JSON key file, the same file will be used to
//     sign URLs when possible.
type Client struct {
	gc        *storage.Client
	projectID string
	// If credsFile is non-empty it will be used for signed URL generation (if possible).
	credsFile string
}

// NewClient creates a new GCS client. It will prefer ADC (Application Default
// Credentials) but if the env var GOOGLE_APPLICATION_CREDENTIALS is set and
// points to a credentials file, that file will be used to create the client
// (and for signed URL operations).
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	var opts []option.ClientOption
	creds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if creds != "" {
		// Use explicit credentials file
		opts = append(opts, option.WithCredentialsFile(creds))
	}

	gc, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("gcs: failed to create storage client: %w", err)
	}

	return &Client{
		gc:        gc,
		projectID: projectID,
		credsFile: creds,
	}, nil
}

// Close closes the underlying GCS client.
func (c *Client) Close() error {
	if c.gc == nil {
		return nil
	}
	return c.gc.Close()
}

// UploadFile uploads the file at localPath to the specified bucket/object.
// It returns a GCS URI of the uploaded object (gs://bucket/object) on success.
func (c *Client) UploadFile(ctx context.Context, bucket, object, localPath string) (string, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("gcs: failed to open local file: %w", err)
	}
	defer f.Close()

	return c.UploadStream(ctx, bucket, object, f)
}

// UploadBytes uploads the provided bytes to bucket/object and returns a gs:// URI.
func (c *Client) UploadBytes(ctx context.Context, bucket, object string, data []byte) (string, error) {
	// Use bytes.NewReader to obtain an io.Reader for the byte slice and forward to UploadStream.
	return c.UploadStream(ctx, bucket, object, bytes.NewReader(data))
}

// UploadStream uploads data from an io.Reader into the given bucket/object.
// Returns the gs:// URI for the object.
func (c *Client) UploadStream(ctx context.Context, bucket, object string, r io.Reader) (string, error) {
	if c.gc == nil {
		return "", errors.New("gcs: client is not initialized")
	}
	wc := c.gc.Bucket(bucket).Object(object).NewWriter(ctx)

	// Consider setting attributes like ContentType, CacheControl, etc. if needed.
	// wc.ContentType = "application/octet-stream"

	if _, err := io.Copy(wc, r); err != nil {
		_ = wc.Close() // best-effort close to free resources
		return "", fmt.Errorf("gcs: failed to write object: %w", err)
	}

	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("gcs: failed to finalize object upload: %w", err)
	}

	// Return the canonical gs:// URL
	return fmt.Sprintf("gs://%s/%s", bucket, object), nil
}

// GenerateSignedURL attempts to create a signed URL for the given object. The
// resulting URL is valid for the provided expiry duration.
//
// Requirements and behavior:
//   - If the CLIENT has a credentials JSON file (GOOGLE_APPLICATION_CREDENTIALS)
//     and it contains a service account private key, this method will sign the URL
//     using that key and return a signed URL.
//   - If signing cannot be performed due to missing credentials or incompatible
//     credential types, an error is returned explaining why.
//
// Note: signing via local key file uses storage.SignedURL which requires the
// service account email and private key.
func (c *Client) GenerateSignedURL(ctx context.Context, bucket, object string, expiry time.Duration) (string, error) {
	if c.credsFile == "" {
		return "", errors.New("gcs: GOOGLE_APPLICATION_CREDENTIALS not set; cannot generate signed URL")
	}

	credsJSON, err := os.ReadFile(c.credsFile)
	if err != nil {
		return "", fmt.Errorf("gcs: failed to read credentials file: %w", err)
	}

	var kd struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
		Type        string `json:"type"`
	}
	if err := json.Unmarshal(credsJSON, &kd); err != nil {
		return "", fmt.Errorf("gcs: failed to parse credentials JSON: %w", err)
	}

	if kd.Type != "service_account" {
		return "", fmt.Errorf("gcs: credentials type is %q, service account key required for SignedURL", kd.Type)
	}
	if kd.ClientEmail == "" || kd.PrivateKey == "" {
		return "", errors.New("gcs: credentials missing client_email or private_key for signing")
	}

	// Use storage.SignedURL to create the signed URL.
	// This requires the bucket to exist and proper permissions on the service account.
	opts := &storage.SignedURLOptions{
		GoogleAccessID: kd.ClientEmail,
		PrivateKey:     []byte(kd.PrivateKey),
		Method:         "GET",
		Expires:        time.Now().Add(expiry),
	}
	u, err := storage.SignedURL(bucket, object, opts)
	if err != nil {
		return "", fmt.Errorf("gcs: failed to generate signed url: %w", err)
	}
	return u, nil
}
