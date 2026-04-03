package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Uploader is a convenience adapter for uploading objects to AWS S3.
//
// Behaviour is gated: by default the adapter will refuse to initialize unless
// either the environment variable ENABLE_S3 is set to "1" or typical AWS
// credential env vars are present. This prevents accidental attempts to use S3
// in environments where credentials are not configured.
//
// Usage example:
//
//	u, err := s3.NewUploader(ctx, \"us-west-2\", \"my-bucket\")
//	if err != nil { ... }
//	url, err := u.UploadFile(ctx, \"path/in/bucket/out.zip\", \"./tmp/out.zip\")
//	if err != nil { ... }
type Uploader struct {
	client   *s3sdk.Client
	uploader *manager.Uploader
	region   string
	bucket   string
}

// NewUploader constructs a new S3 uploader.
//
// The function loads AWS configuration using the SDK's default chain. It will
// refuse to initialize unless either:
//   - the environment variable ENABLE_S3 is set to "1", or
//   - AWS_ACCESS_KEY_ID is present in the environment (or other credential sources
//     are available to the SDK).
//
// Provide an empty region to let the SDK pick a default region if available.
func NewUploader(ctx context.Context, region string, bucket string) (*Uploader, error) {
	if bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	// gating: require explicit opt-in or AWS creds present
	enable := os.Getenv("ENABLE_S3")
	awsKey := os.Getenv("AWS_ACCESS_KEY_ID")
	if enable != "1" && awsKey == "" {
		return nil, errors.New("S3 uploads are disabled: set ENABLE_S3=1 or provide AWS credentials")
	}

	// Load SDK config (this will use env vars, shared config, or instance profile)
	var cfgOpts []func(*config.LoadOptions) error
	if region != "" {
		cfgOpts = append(cfgOpts, config.WithRegion(region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3sdk.NewFromConfig(cfg)
	u := manager.NewUploader(client)

	return &Uploader{
		client:   client,
		uploader: u,
		region:   cfg.Region,
		bucket:   bucket,
	}, nil
}

// UploadFile uploads the file located at filePath to key (object key) in the
// configured bucket. It returns a public-ish URL (best-effort) to the uploaded object.
//
// Note: object ACLs/permissions are not changed by this helper; if you need an
// object to be publicly accessible you must configure bucket policy or specify
// the appropriate ACL in a subsequent call. Here we set ContentType to
// application/octet-stream by default.
func (u *Uploader) UploadFile(ctx context.Context, key string, filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat file: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory: %s", filePath)
	}

	// infer content type from extension (basic)
	contentType := detectContentType(filePath)

	input := &s3sdk.PutObjectInput{
		Bucket:      &u.bucket,
		Key:         &key,
		Body:        f,
		ContentType: &contentType,
		// Set a reasonable storage class; caller may override in future enhancements.
		StorageClass: types.StorageClassStandard,
	}

	// Use uploader to stream large files efficiently
	_, err = u.uploader.Upload(ctx, input)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	return u.objectURL(key), nil
}

// UploadBytes uploads the provided bytes as an object with the given key.
// contentType may be empty, in which case application/octet-stream is used.
func (u *Uploader) UploadBytes(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	input := &s3sdk.PutObjectInput{
		Bucket:       &u.bucket,
		Key:          &key,
		Body:         bytesReader(data),
		ContentType:  &contentType,
		StorageClass: types.StorageClassStandard,
	}

	_, err := u.uploader.Upload(ctx, input)
	if err != nil {
		return "", fmt.Errorf("upload bytes failed: %w", err)
	}
	return u.objectURL(key), nil
}

// objectURL builds a best-effort HTTP URL for the uploaded object. It does not
// guarantee the object is publicly accessible; it merely provides the standard
// S3 endpoint location for the object.
func (u *Uploader) objectURL(key string) string {
	// Use virtual-hosted-style URL if region is known; fallback to path-style.
	escapedKey := url.PathEscape(key)
	if u.region != "" {
		// Example: https://bucket.s3.us-west-2.amazonaws.com/key
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", u.bucket, u.region, escapedKey)
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", u.bucket, escapedKey)
}

// detectContentType performs a very small heuristic based on file extension.
func detectContentType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".zip":
		return "application/zip"
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".csv":
		return "text/csv"
	default:
		return "application/octet-stream"
	}
}

// bytesReader returns an io.ReadSeekCloser backed by the provided byte slice.
// manager.Uploader accepts io.Reader but providing a ReadSeeker can allow
// retries; the uploader accepts io.Reader too. We implement a simple reader.
func bytesReader(b []byte) io.Reader {
	return bytes.NewReader(b)
}

// --- small helper for uploading with a contextual timeout (optional) ---

// UploadFileWithTimeout uploads a file but enforces the provided timeout.
func (u *Uploader) UploadFileWithTimeout(parent context.Context, key, filePath string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	return u.UploadFile(ctx, key, filePath)
}
