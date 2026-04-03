package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Uploader defines an interface for uploading and managing files in a storage backend.
type Uploader interface {
	// Upload copies a file from localPath to destination path within the backend.
	// It returns a URL (or link) where the uploaded file can be accessed and/or an error.
	Upload(ctx context.Context, localPath string, destPath string) (string, error)

	// UploadBytes writes the provided byte slice to destPath in the backend and returns a URL.
	UploadBytes(ctx context.Context, data []byte, destPath string) (string, error)

	// Delete removes the file at destPath from the backend.
	Delete(ctx context.Context, destPath string) error

	// URL returns a URL or identifier for the file located at destPath within the backend.
	URL(destPath string) string
}

// LocalUploader is a simple implementation of Uploader that stores files on the local filesystem.
// The Root directory is treated as the storage root; destPath is resolved relative to Root.
type LocalUploader struct {
	Root string // root directory on disk where files will be stored
}

// NewLocalUploader creates a new LocalUploader which stores files under the given root directory.
// If root is empty, it defaults to "./tmp/storage".
func NewLocalUploader(root string) *LocalUploader {
	if root == "" {
		root = filepath.Join(".", "tmp", "storage")
	}
	return &LocalUploader{Root: root}
}

// ensureTargetDir ensures that the directory for the given destination path exists.
func (l *LocalUploader) ensureTargetDir(destPath string) error {
	targetDir := filepath.Dir(filepath.Join(l.Root, destPath))
	return os.MkdirAll(targetDir, 0o755)
}

// Upload copies a local file into the storage root under destPath and returns a file:// URL.
func (l *LocalUploader) Upload(ctx context.Context, localPath string, destPath string) (string, error) {
	// Check context
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Ensure source file exists
	srcFile, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("open source file: %w", err)
	}
	defer srcFile.Close()

	if err := l.ensureTargetDir(destPath); err != nil {
		return "", fmt.Errorf("ensure target dir: %w", err)
	}

	destFull := filepath.Join(l.Root, destPath)
	// Create destination file (truncate if exists)
	dstFile, err := os.Create(destFull)
	if err != nil {
		return "", fmt.Errorf("create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return "", fmt.Errorf("copy file contents: %w", err)
	}

	abs, err := filepath.Abs(destFull)
	if err != nil {
		abs = destFull // fallback to relative path
	}

	return "file://" + abs, nil
}

// UploadBytes writes data to destPath under the storage root and returns a file:// URL.
func (l *LocalUploader) UploadBytes(ctx context.Context, data []byte, destPath string) (string, error) {
	// Check context
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	if err := l.ensureTargetDir(destPath); err != nil {
		return "", fmt.Errorf("ensure target dir: %w", err)
	}

	destFull := filepath.Join(l.Root, destPath)
	if err := os.WriteFile(destFull, data, 0o644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	abs, err := filepath.Abs(destFull)
	if err != nil {
		abs = destFull
	}

	return "file://" + abs, nil
}

// Delete removes the file at destPath within the storage root.
func (l *LocalUploader) Delete(ctx context.Context, destPath string) error {
	// Check context
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	destFull := filepath.Join(l.Root, destPath)
	if err := os.Remove(destFull); err != nil {
		if os.IsNotExist(err) {
			return nil // idempotent delete
		}
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

// URL returns a file:// URL for the path under the storage root.
// This does not validate that the file exists.
func (l *LocalUploader) URL(destPath string) string {
	destFull := filepath.Join(l.Root, destPath)
	abs, err := filepath.Abs(destFull)
	if err != nil {
		// fallback to joined path if Abs fails
		return "file://" + destFull
	}
	return "file://" + abs
}
