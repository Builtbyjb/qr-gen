package local

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

// Adapter is a small convenience wrapper that exposes a simple Upload(ctx, localPath)
// method used by higher-level orchestration code. It stores uploaded files under
// a configurable relative destination directory inside the storage root.
type Adapter struct {
	// Store is the underlying filesystem-backed storage implementation.
	// It is created via New(...) and must not be nil.
	Store *LocalStorage

	// DestDir is the relative directory (under the storage root) where uploaded
	// files will be placed. Example: "artifacts" or "uploads".
	DestDir string
}

// NewAdapter creates a new Adapter. `basePath` configures where the storage root will live
// on disk (if empty a sensible default is used by New). `destDir` is the relative directory
// inside that storage root where uploaded files will be placed; if empty it defaults to "artifacts".
func NewAdapter(basePath string, destDir string) (*Adapter, error) {
	store, err := New(basePath)
	if err != nil {
		return nil, fmt.Errorf("creating local storage: %w", err)
	}
	if destDir == "" {
		destDir = "artifacts"
	}
	return &Adapter{
		Store:   store,
		DestDir: destDir,
	}, nil
}

// Upload implements the package-level uploader contract used by the service layer.
// It copies the provided localPath into the storage root under DestDir with a
// timestamped filename to avoid collisions and returns a file:// URL to the stored file.
func (a *Adapter) Upload(ctx context.Context, localPath string) (string, error) {
	if a == nil || a.Store == nil {
		return "", fmt.Errorf("local adapter is not initialized")
	}
	if localPath == "" {
		return "", fmt.Errorf("localPath is required")
	}

	filename := filepath.Base(localPath)
	dest := filepath.Join(a.DestDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), filename))

	// Use the underlying LocalStorage.UploadFile which expects a local src path and a relative dest path.
	return a.Store.UploadFile(ctx, localPath, dest)
}

// Compile-time check: Adapter implements a simplified uploader interface (Upload only).
// We assert against an anonymous interface with the Upload method to avoid requiring
// unrelated methods (Delete, etc.) from the package-level storage interface.
var _ interface {
	Upload(context.Context, string) (string, error)
} = (*Adapter)(nil)
