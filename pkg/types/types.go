package types

import (
	"fmt"
	"strings"
)

// Format represents supported output formats and exports names expected by other packages.
type Format int

const (
	PDF Format = iota
	PNG
	SVG
	JPEG
)

// Aliases for external package and test compatibility.
const (
	FormatPDF  = PDF
	FormatPNG  = PNG
	FormatSVG  = SVG
	FormatJPEG = JPEG
)

func (f Format) String() string {
	switch f {
	case PDF:
		return "pdf"
	case PNG:
		return "png"
	case SVG:
		return "svg"
	case JPEG:
		return "jpeg"
	default:
		return "unknown"
	}
}

// FormatFromString converts a user provided string into a Format value.
// It is case-insensitive and accepts common aliases (e.g. "jpg" -> jpeg).
func FormatFromString(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pdf":
		return PDF, nil
	case "png":
		return PNG, nil
	case "svg":
		return SVG, nil
	case "jpeg", "jpg":
		return JPEG, nil
	default:
		return PDF, fmt.Errorf("invalid format: %q", s)
	}
}

// Storage represents supported storage backends and uses exported constant names
// that match the rest of the codebase.
type Storage int

const (
	LOCAL Storage = iota
	AWS_S3
	GCS
	AZURE_BLOB
	GOOGLE_DRIVE
	DROPBOX
	ONE_DRIVE
)

// Aliases for external package and test compatibility.
const (
	StorageLocal       = LOCAL
	StorageS3          = AWS_S3
	StorageGCS         = GCS
	StorageAzure       = AZURE_BLOB
	StorageGoogleDrive = GOOGLE_DRIVE
	StorageDropbox     = DROPBOX
	StorageOneDrive    = ONE_DRIVE
)

func (s Storage) String() string {
	switch s {
	case LOCAL:
		return "local"
	case AWS_S3:
		return "s3"
	case GCS:
		return "gcs"
	case AZURE_BLOB:
		return "azure"
	case GOOGLE_DRIVE:
		return "gdrive"
	case DROPBOX:
		return "dropbox"
	case ONE_DRIVE:
		return "onedrive"
	default:
		return "unknown"
	}
}

// StorageFromString converts a string into a Storage value.
// Accepts a few common aliases for each backend.
func StorageFromString(str string) (Storage, error) {
	switch strings.ToLower(strings.TrimSpace(str)) {
	case "local", "disk", "filesystem", "":
		return LOCAL, nil
	case "s3", "aws", "aws_s3":
		return AWS_S3, nil
	case "gcs", "google_cloud_storage", "google", "gcloud":
		return GCS, nil
	case "azure", "azure_blob", "azure_blob_storage":
		return AZURE_BLOB, nil
	case "google_drive", "gdrive":
		return GOOGLE_DRIVE, nil
	case "dropbox":
		return DROPBOX, nil
	case "one_drive", "onedrive":
		return ONE_DRIVE, nil
	default:
		return LOCAL, fmt.Errorf("invalid storage option: %q", str)
	}
}

// Argument contains the parsed CLI/input arguments for QR generation.
// Extended to include fields used by service/storage/email.
type Argument struct {
	Quantity  int     `json:"quantity"`
	Info      string  `json:"info"`
	Size      int     `json:"size"` // canvas size / image size in pixels
	URL       string  `json:"url"`
	Format    Format  `json:"format"`
	Storage   Storage `json:"storage"`
	ChunkSize int     `json:"chunk_size"`

	// Storage / cloud fields
	ProjectID string `json:"project_id"` // for GCS
	Bucket    string `json:"bucket"`     // storage bucket name

	// Email options
	SendEmail bool   `json:"send_email"`
	EmailTo   string `json:"email_to"`
}

// Default values used when fields are not provided.
const (
	DefaultSize      = 50
	DefaultChunkSize = 500
)

// Validate checks required fields and basic constraints. It returns an error when
// arguments are invalid.
func (a *Argument) Validate() error {
	if a == nil {
		return fmt.Errorf("argument is nil")
	}
	if a.Quantity <= 0 {
		return fmt.Errorf("quantity must be greater than 0")
	}
	if strings.TrimSpace(a.URL) == "" {
		return fmt.Errorf("url is required")
	}
	// Size sanity check
	if a.Size <= 0 {
		a.Size = DefaultSize
	}
	if a.ChunkSize <= 0 {
		a.ChunkSize = DefaultChunkSize
	}
	return nil
}

// String returns a concise representation of the Argument.
func (a Argument) String() string {
	return fmt.Sprintf("Argument{Quantity:%d, Info:%q, Size:%d, URL:%q, Format:%s, Storage:%s, ChunkSize:%d, ProjectID:%q, Bucket:%q, SendEmail:%t, EmailTo:%q}",
		a.Quantity, a.Info, a.Size, a.URL, a.Format.String(), a.Storage.String(), a.ChunkSize, a.ProjectID, a.Bucket, a.SendEmail, a.EmailTo)
}
