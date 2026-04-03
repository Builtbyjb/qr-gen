package util

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ZipFolders compresses the provided folder paths into a single zip archive.
//   - outputDir: directory where the zip will be created (created if missing).
//   - zipName: the filename for the zip (e.g. "archive.zip"). If empty, a timestamped name will be generated.
//   - folders: slice of folder paths to include in the archive. Each folder's contents
//     will be stored under a top-level directory named after the folder's base name.
//
// Returns the full path to the created zip file or an error.
func ZipFolders(outputDir, zipName string, folders []string) (string, error) {
	if len(folders) == 0 {
		return "", fmt.Errorf("no folders provided to zip")
	}

	// Ensure output directory exists
	if outputDir == "" {
		outputDir = "."
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	if strings.TrimSpace(zipName) == "" {
		zipName = fmt.Sprintf("archive_%d.zip", time.Now().UnixMilli())
	}

	zipPath := filepath.Join(outputDir, zipName)
	outFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file %s: %w", zipPath, err)
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer func() {
		// Ensure the writer is closed; if it errors, prefer to return original error or log
		_ = zipWriter.Close()
	}()

	for _, folder := range folders {
		absFolder, err := filepath.Abs(folder)
		if err != nil {
			return "", fmt.Errorf("failed to resolve folder path %s: %w", folder, err)
		}
		info, err := os.Stat(absFolder)
		if err != nil {
			return "", fmt.Errorf("failed to stat folder %s: %w", folder, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("path is not a directory: %s", folder)
		}

		baseName := filepath.Base(absFolder)

		// Walk the folder and add files
		err = filepath.WalkDir(absFolder, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			// Skip directories: we create entries for files only. Directories are implied.
			if d.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(absFolder, path)
			if err != nil {
				return err
			}
			// Normalize to forward slashes for zip spec
			zipEntryName := filepath.ToSlash(filepath.Join(baseName, relPath))

			if err := addFileToZip(zipWriter, path, zipEntryName); err != nil {
				return fmt.Errorf("failed to add file %s to zip: %w", path, err)
			}
			return nil
		})
		if err != nil {
			return "", err
		}
	}

	// Close the zip writer to flush the archive
	if err := zipWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to finalize zip file: %w", err)
	}

	return zipPath, nil
}

// ZipFiles compresses the provided file paths into a single zip archive.
// Files will be added at the archive root using their base filenames unless a
// zipEntryName is provided via the optional map parameter (entryNames[filePath] = zipName).
// - outputPath: full path (including filename) for the zip file to create.
// - files: slice of absolute or relative file paths to include.
// - entryNames: optional map mapping original file paths to desired entry names inside the zip.
// Returns the created zip path or an error.
func ZipFiles(outputPath string, files []string, entryNames map[string]string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files provided to zip")
	}
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file %s: %w", outputPath, err)
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer func() {
		_ = zipWriter.Close()
	}()

	for _, fpath := range files {
		info, err := os.Stat(fpath)
		if err != nil {
			return "", fmt.Errorf("failed to stat file %s: %w", fpath, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("expected file but found directory: %s", fpath)
		}

		entryName := entryNames[fpath]
		if entryName == "" {
			entryName = filepath.Base(fpath)
		}
		entryName = filepath.ToSlash(entryName)

		if err := addFileToZip(zipWriter, fpath, entryName); err != nil {
			return "", fmt.Errorf("failed to add file %s to zip: %w", fpath, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to finalize zip file: %w", err)
	}

	return outputPath, nil
}

// addFileToZip appends a single file into the provided zip.Writer with the specified entry name.
func addFileToZip(zw *zip.Writer, filePath, entryName string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	// Create a header based on file info
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = entryName
	// Use Deflate compression for non-empty files
	header.Method = zip.Deflate
	// Preserve modification time
	header.Modified = info.ModTime()

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(writer, file)
	return err
}
