package service

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/yourusername/qrgen/pkg/codegen"
	"github.com/yourusername/qrgen/pkg/db"
	"github.com/yourusername/qrgen/pkg/oauth"
	"github.com/yourusername/qrgen/pkg/pdf"
	localadapter "github.com/yourusername/qrgen/pkg/storage/local"
	"github.com/yourusername/qrgen/pkg/types"
)

// StorageUploader uploads a local artifact and returns a publicly-accessible URL (or identifier).
type StorageUploader interface {
	Upload(ctx context.Context, localPath string) (string, error)
}

// DBRepository persists generated QR codes and related metadata.
type DBRepository interface {
	Init(ctx context.Context) error
	SaveCodes(ctx context.Context, codes []string) error
	Close(ctx context.Context) error
}

// EmailSender sends notifications (optionally with attachments).
type EmailSender interface {
	SendWithAttachment(ctx context.Context, to string, subject string, body string, attachmentPath string) error
}

// Service orchestrates QR code generation, PDF creation, zipping, persistence and upload.
type Service struct {
	Arg      *types.Argument
	uploader StorageUploader
	repo     DBRepository
	email    EmailSender

	// runtime dirs
	tmpBase   string
	pdfFolder string
	zipFolder string
	csvPath   string
}

// New constructs a Service with reasonable defaults. It will attempt to initialize
// storage and DB adapters based on the provided argument. If adapters cannot be
// created, they will be nil and the service will continue where reasonable.
func New(arg *types.Argument) *Service {
	// Construct the service and wire default local uploader and optional Gmail sender.
	svc := &Service{
		Arg:     arg,
		tmpBase: filepath.Join("tmp"),
	}

	// Initialize a local storage adapter (writes artifacts under a local storage root).
	// Now that local.New accepts an empty basePath and defaults to tmp/storage, this
	// will always succeed unless there is a filesystem permission error.
	if adapter, err := localadapter.NewAdapter("", "artifacts"); err == nil {
		svc.uploader = adapter
	}

	// Attempt to wire a DB repository from environment variables (no-op when absent).
	if repo, err := db.NewRepositoryFromEnv(); err == nil && repo != nil {
		svc.repo = repo
	}

	// Attempt to wire a Gmail sender from OAuth credentials/token (no-op when absent).
	if sender, err := oauth.NewGmailSenderFromEnv(context.Background()); err == nil && sender != nil {
		svc.email = sender
	}

	return svc
}

// Run executes the full pipeline. Steps:
//   - validate args
//   - generate codes
//   - persist to DB (optional)
//   - write CSV
//   - render PDFs (chunked), produce per-chunk folder(s)
//   - zip the PDF folders into a single zip
//   - upload zip to storage (optional)
//   - send email with attachment or link (optional)
//   - cleanup temporary artifacts (unless DEBUG / TEST enabled)
func (s *Service) Run(ctx context.Context) error {
	if s.Arg == nil {
		return errors.New("nil argument provided")
	}
	if err := s.Arg.Validate(); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}

	// Prepare tmp dirs
	ts := time.Now().UnixMilli()
	s.pdfFolder = filepath.Join(s.tmpBase, "pdfs", fmt.Sprintf("%d", ts))
	s.zipFolder = filepath.Join(s.tmpBase, "zips")
	if err := os.MkdirAll(s.pdfFolder, 0o755); err != nil {
		return fmt.Errorf("failed to create pdf folder: %w", err)
	}
	if err := os.MkdirAll(s.zipFolder, 0o755); err != nil {
		return fmt.Errorf("failed to create zip folder: %w", err)
	}

	// 1) Generate codes
	codes := make([]string, 0, s.Arg.Quantity)
	for i := 0; i < s.Arg.Quantity; i++ {
		c := codegen.GenerateQRCode(int64(i))
		codes = append(codes, c)
	}

	// 2) Validate uniqueness
	if !uniqueSlice(codes) {
		return errors.New("duplicate QR codes generated")
	}

	// 3) Persist to DB (optional)
	if s.repo != nil {
		if err := s.repo.Init(ctx); err != nil {
			return fmt.Errorf("failed to init db repo: %w", err)
		}
		defer s.repo.Close(ctx)
		if err := s.repo.SaveCodes(ctx, codes); err != nil {
			// Non-fatal: we surface the error but continue to allow downstream operations.
			return fmt.Errorf("failed to save codes to db: %w", err)
		}
	}

	// 4) Write CSV of codes
	csvPath, err := s.writeCSV(codes)
	if err != nil {
		return fmt.Errorf("failed to write csv: %w", err)
	}
	s.csvPath = csvPath

	// 5) Generate PDFs (chunked)
	chunkSize := 500
	if s.Arg.ChunkSize > 0 {
		chunkSize = s.Arg.ChunkSize
	}
	folderPaths, err := s.generatePDFsConcurrently(codes, chunkSize)
	if err != nil {
		return fmt.Errorf("failed to generate PDFs: %w", err)
	}

	// 6) Zip the folder(s)
	zipFile := filepath.Join(s.zipFolder, fmt.Sprintf("qr_codes_%d.zip", ts))
	if err := zipFolders(zipFile, folderPaths); err != nil {
		return fmt.Errorf("failed to create zip: %w", err)
	}

	// 7) Upload zip (optional)
	var uploadedURL string
	if s.uploader != nil {
		uploadedURL, err = s.uploader.Upload(ctx, zipFile)
		if err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}
	}

	// 8) Send email (optional)
	if s.Arg.SendEmail && s.email != nil {
		to := s.Arg.EmailTo
		if to == "" {
			return errors.New("email requested but --email-to not provided")
		}
		subject := "Your QR Codes"
		body := "Your QR codes are ready."
		if uploadedURL != "" {
			body = fmt.Sprintf("%s\n\nDownload: %s", body, uploadedURL)
		}
		if err := s.email.SendWithAttachment(ctx, to, subject, body, zipFile); err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	// 9) Cleanup: match Java behaviour — the CSV and PDF folders are intentionally
	// preserved for downstream use / inspection. The zip is removed only after a
	// successful upload (DEBUG=0 equivalent).
	if uploadedURL != "" {
		_ = os.Remove(zipFile)
	}

	return nil
}

// writeCSV writes the codes to a timestamped CSV under tmp/csv and returns the file path.
func (s *Service) writeCSV(codes []string) (string, error) {
	dir := filepath.Join(s.tmpBase, "csv")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	fp := filepath.Join(dir, fmt.Sprintf("qr_codes_%d.csv", time.Now().UnixMilli()))
	f, err := os.Create(fp)
	if err != nil {
		return "", err
	}
	defer f.Close()

	for _, c := range codes {
		if _, err := f.WriteString(c + "\n"); err != nil {
			return "", err
		}
	}
	return fp, nil
}

// generatePDFsConcurrently splits codes into chunks and generates PDFs concurrently.
// It returns the folder paths that contain generated PDFs.
func (s *Service) generatePDFsConcurrently(codes []string, chunkSize int) ([]string, error) {
	if chunkSize <= 0 {
		chunkSize = 500
	}
	var chunks [][]string
	for i := 0; i < len(codes); i += chunkSize {
		end := i + chunkSize
		if end > len(codes) {
			end = len(codes)
		}
		chunks = append(chunks, codes[i:end])
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(chunks))
	folderPaths := make([]string, 0, len(chunks))
	mu := sync.Mutex{}

	for idx, chunk := range chunks {
		wg.Add(1)
		go func(i int, cks []string) {
			defer wg.Done()
			folderName := filepath.Join(s.pdfFolder, fmt.Sprintf("batch_%d", i))
			if err := os.MkdirAll(folderName, 0o755); err != nil {
				errCh <- fmt.Errorf("failed to create folder %s: %w", folderName, err)
				return
			}

			// pdf.GeneratePDF(folder, idx, codes, arg) - call with folder first.
			// We ignore the returned PDF path here because we collect folder names.
			if _, err := pdf.GeneratePDF(folderName, i, cks, s.Arg); err != nil {
				errCh <- fmt.Errorf("pdf generation error for batch %d: %w", i, err)
				return
			}

			mu.Lock()
			folderPaths = append(folderPaths, folderName)
			mu.Unlock()
		}(idx, chunk)
	}

	wg.Wait()
	close(errCh)
	// If any error occurred return the first
	for e := range errCh {
		return nil, e
	}

	return folderPaths, nil
}

// zipFolders compresses the provided folders into a single zip file at destZip.
func zipFolders(destZip string, folders []string) error {
	fzip, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer fzip.Close()

	zw := zip.NewWriter(fzip)
	defer zw.Close()

	for _, folder := range folders {
		err := filepath.Walk(folder, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(filepath.Dir(folder), path)
			if err != nil {
				rel = info.Name()
			}
			zipPath := filepath.Join(filepath.Base(folder), rel)
			fw, err := zw.Create(zipPath)
			if err != nil {
				return err
			}
			fr, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fr.Close()
			if _, err := io.Copy(fw, fr); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("zipping folder %s failed: %w", folder, err)
		}
	}
	return nil
}

// uniqueSlice returns true when all elements are unique.
func uniqueSlice(arr []string) bool {
	seen := make(map[string]struct{}, len(arr))
	for _, v := range arr {
		if _, ok := seen[v]; ok {
			return false
		}
		seen[v] = struct{}{}
	}
	return true
}

// Below are small helpers. Adapters and senders are optional; failures to initialize are non-fatal.
//
// We intentionally avoid initializing heavy external dependencies (Gmail, cloud SDKs)
// automatically in the service package. Callers may create and attach adapters or
// senders as needed.

// NewLocalUploader creates a local uploader adapter rooted at basePath.
// Returns nil if creation fails.
func NewLocalUploader(basePath string) StorageUploader {
	if a, err := localadapter.NewAdapter(basePath, "artifacts"); err == nil {
		return a
	}
	return nil
}

// NewGCSUploader is intentionally not provided here to avoid adding GCS SDK initialization
// in the core service package. Create a GCS uploader explicitly where you have credentials.
//
// NewS3Uploader is intentionally omitted here for the same reason.
