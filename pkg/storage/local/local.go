package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalStorage is a simple filesystem-backed storage adapter.
// All operations are rooted at BasePath. Paths provided to methods
// are treated as relative to BasePath. The adapter protects against
// path traversal outside BasePath.
type LocalStorage struct {
	BasePath string
}

// New creates a LocalStorage rooted at basePath. The base directory
// will be created if it does not already exist.
func New(basePath string) (*LocalStorage, error) {
	if strings.TrimSpace(basePath) == "" {
		basePath = filepath.Join("tmp", "storage")
	}

	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("getting absolute base path: %w", err)
	}

	if err := os.MkdirAll(absBase, 0o755); err != nil {
		return nil, fmt.Errorf("creating base path: %w", err)
	}

	return &LocalStorage{BasePath: absBase}, nil
}

// UploadFile copies a file from srcPath (local filesystem) into the storage
// under destRelPath (relative to BasePath). It writes atomically using a temp
// file + rename and returns a file:// URL to the stored file.
func (s *LocalStorage) UploadFile(ctx context.Context, srcPath, destRelPath string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("open source file: %w", err)
	}
	defer src.Close()

	destAbs, err := s.safeAbsPath(destRelPath)
	if err != nil {
		return "", err
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destAbs), 0o755); err != nil {
		return "", fmt.Errorf("create dest dir: %w", err)
	}

	// Create a temp file in the destination dir and copy, then rename to final
	tmpFile, err := os.CreateTemp(filepath.Dir(destAbs), ".tmp-*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		// best-effort cleanup on error
		if _, statErr := os.Stat(tmpName); statErr == nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := io.Copy(tmpFile, src); err != nil {
		return "", fmt.Errorf("copy data: %w", err)
	}

	// Flush and close before rename
	if err := tmpFile.Sync(); err != nil {
		return "", fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, destAbs); err != nil {
		return "", fmt.Errorf("rename temp to dest: %w", err)
	}

	return s.GetURL(destRelPath)
}

// UploadBytes writes the provided bytes into destRelPath atomically and returns a file:// URL.
func (s *LocalStorage) UploadBytes(ctx context.Context, data []byte, destRelPath string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	destAbs, err := s.safeAbsPath(destRelPath)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(destAbs), 0o755); err != nil {
		return "", fmt.Errorf("create dest dir: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(destAbs), ".tmp-*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return "", fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, destAbs); err != nil {
		return "", fmt.Errorf("rename temp to dest: %w", err)
	}

	return s.GetURL(destRelPath)
}

// ReadFile reads the file at relPath (relative to BasePath) and returns its contents.
func (s *LocalStorage) ReadFile(ctx context.Context, relPath string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	abs, err := s.safeAbsPath(relPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return data, nil
}

// Exists returns true if the file at relPath exists.
func (s *LocalStorage) Exists(ctx context.Context, relPath string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	abs, err := s.safeAbsPath(relPath)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(abs)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("stat file: %w", err)
}

// Delete removes the file at relPath.
func (s *LocalStorage) Delete(ctx context.Context, relPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	abs, err := s.safeAbsPath(relPath)
	if err != nil {
		return err
	}

	if err := os.Remove(abs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

// List returns a list of files under the directory prefix (relative to BasePath).
// The returned paths are relative to BasePath and use forward slashes.
func (s *LocalStorage) List(ctx context.Context, prefix string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	startAbs, err := s.safeAbsPath(prefix)
	if err != nil {
		return nil, err
	}

	var results []string
	err = filepath.WalkDir(startAbs, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			// If a single file is unreadable, continue walking
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(s.BasePath, p)
		if err != nil {
			return nil
		}
		// Normalize to forward slashes for returned paths
		results = append(results, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walkdir: %w", err)
	}
	return results, nil
}

// GetURL returns a file:// URL that points to the stored file.
func (s *LocalStorage) GetURL(relPath string) (string, error) {
	abs, err := s.safeAbsPath(relPath)
	if err != nil {
		return "", err
	}
	u := url.URL{
		Scheme: "file",
		Path:   abs,
	}
	return u.String(), nil
}

// safeAbsPath returns the absolute filesystem path for relPath ensuring that
// the resolved path is inside BasePath. relPath may contain subdirectories.
// It returns an error if the resolved path would escape BasePath.
func (s *LocalStorage) safeAbsPath(relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		// don't allow absolute destination paths to escape base
		return "", fmt.Errorf("destination path must be relative")
	}
	clean := filepath.Clean(relPath)
	// Prevent paths that attempt to traverse outside the base via ".."
	if strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || clean == ".." {
		return "", fmt.Errorf("relative path escapes base: %s", relPath)
	}
	joined := filepath.Join(s.BasePath, clean)
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("abs path: %w", err)
	}

	// Verify absJoined is within BasePath
	base := s.BasePath
	// Ensure both have path separators at end to avoid prefix collisions (e.g. /tmp/base and /tmp/base2)
	baseWithSep := strings.TrimRight(base, string(os.PathSeparator)) + string(os.PathSeparator)
	absWithSep := strings.TrimRight(absJoined, string(os.PathSeparator)) + string(os.PathSeparator)
	if !strings.HasPrefix(absWithSep, baseWithSep) {
		return "", fmt.Errorf("resolved path is outside the base path")
	}
	return absJoined, nil
}

// CreateDir creates a directory (and parents) under the base path.
func (s *LocalStorage) CreateDir(ctx context.Context, relDir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	abs, err := s.safeAbsPath(relDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return fmt.Errorf("mkdirall: %w", err)
	}
	return nil
}

// TempFile returns a path to a temporary file placed under the provided relDir
// inside the storage base path. Caller is responsible for removing the file.
func (s *LocalStorage) TempFile(ctx context.Context, relDir, pattern string) (string, *os.File, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	absDir, err := s.safeAbsPath(relDir)
	if err != nil {
		return "", nil, err
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("mkdirall: %w", err)
	}
	f, err := os.CreateTemp(absDir, pattern)
	if err != nil {
		return "", nil, fmt.Errorf("create temp: %w", err)
	}
	return f.Name(), f, nil
}

// ModTime returns the modification time for a given relative path.
func (s *LocalStorage) ModTime(ctx context.Context, relPath string) (time.Time, error) {
	if err := ctx.Err(); err != nil {
		return time.Time{}, err
	}
	abs, err := s.safeAbsPath(relPath)
	if err != nil {
		return time.Time{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return time.Time{}, fmt.Errorf("stat: %w", err)
	}
	return info.ModTime(), nil
}
