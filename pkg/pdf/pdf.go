package pdf

import (
	"bytes"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jung-kurt/gofpdf"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/yourusername/qrgen/pkg/types"
)

// GeneratePDF creates a PDF file containing the provided QR code strings.
// Each code will be encoded as a QR image (PNG) with the provided base URL
// (arg.URL is prepended to each code when encoding) and rendered into the PDF.
// The generated PDF is written to folder with filename "qr_codes_<idx>.pdf".
// It returns the full path to the created PDF.
func GeneratePDF(folder string, idx int, codes []string, arg *types.Argument) (string, error) {
	if arg == nil {
		return "", fmt.Errorf("argument cannot be nil")
	}

	// Ensure output folder exists
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return "", fmt.Errorf("failed to create folder %s: %w", folder, err)
	}

	filename := fmt.Sprintf("qr_codes_%d.pdf", idx)
	outPath := filepath.Join(folder, filename)

	// Create new PDF (A4 portrait)
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("QR Code", false)
	pdf.SetAuthor("qrgen", false)
	pdf.SetAutoPageBreak(true, 15)

	// Layout: 2 columns x 2 rows per page (4 images per page).
	// These values were chosen to produce reasonably sized QR images on A4.
	cols := 2
	rows := 2
	perPage := cols * rows

	// Determine image size (mm). Use a moderate size. If arg.Size is large,
	// scale proportionally but keep reasonable bounds.
	pageW, pageH := pdf.GetPageSize()
	margin := 15.0
	usableW := pageW - margin*2
	_ = pageH // pageH available for future layout adjustments

	imgW := usableW / float64(cols) * 0.9 // 90% of the column width
	imgH := imgW                          // square
	xOffset := (usableW/float64(cols) - imgW) / 2
	yOffset := 10.0 // spacing from top inside margin

	// Font setup for labels
	pdf.SetFont("Arial", "", 10)

	// Iterate codes in pages
	for i := 0; i < len(codes); i += perPage {
		pdf.AddPage()

		// Header (centered title)
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 10, "QR Codes", "", 1, "C", false, 0, "")
		pdf.Ln(2)
		pdf.SetFont("Arial", "", 10)

		pageSliceEnd := i + perPage
		if pageSliceEnd > len(codes) {
			pageSliceEnd = len(codes)
		}
		pageCodes := codes[i:pageSliceEnd]

		for pi, code := range pageCodes {
			// Grid position
			col := pi % cols
			row := pi / cols

			// Compute coordinates
			x := margin + float64(col)*(usableW/float64(cols)) + xOffset
			y := margin + yOffset + float64(row)*(imgH+20) // 20mm vertical spacing for label

			// Generate QR image bytes
			fullURL := strings.TrimRight(arg.URL, "/") + code
			imgBytes, err := generateQRCodePNG(fullURL, arg.Size)
			if err != nil {
				return "", fmt.Errorf("failed to generate qr for %s: %w", code, err)
			}

			// Register image with a unique name for this PDF instance
			imgName := fmt.Sprintf("img_%d_%d_%d.png", idx, i, pi)
			imgOpt := gofpdf.ImageOptions{
				ImageType: "PNG",
				ReadDpi:   false,
			}
			// gofpdf.RegisterImageOptionsReader reads the image from an io.Reader.
			if err := registerImageFromBytes(pdf, imgName, imgOpt, imgBytes); err != nil {
				return "", fmt.Errorf("failed to register image %s: %w", imgName, err)
			}

			// Draw the image
			pdf.ImageOptions(imgName, x, y, imgW, imgH, false, imgOpt, 0, "")

			// Draw label centered below the image
			labelY := y + imgH + 4
			pdf.SetXY(x, labelY)
			pdf.SetFont("Arial", "", 9)
			// limit label length to avoid overflow
			label := code
			if len(label) > 40 {
				label = label[:37] + "..."
			}
			cellW := imgW
			pdf.CellFormat(cellW, 6, label, "", 0, "C", false, 0, "")
		}
	}

	// Save file
	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return "", fmt.Errorf("failed to write pdf %s: %w", outPath, err)
	}

	return outPath, nil
}

// GeneratePDFs creates multiple PDF files by iterating over chunks of codes.
// It returns a slice of generated PDF file paths.
func GeneratePDFs(folder string, chunks [][]string, arg *types.Argument) ([]string, error) {
	var paths []string
	for idx, chunk := range chunks {
		p, err := GeneratePDF(folder, idx, chunk, arg)
		if err != nil {
			return paths, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// generateQRCodePNG produces PNG bytes for the provided content (URL + code).
// size parameter is interpreted as the desired canvas size in pixels (higher
// values produce higher resolution images). If size <= 0 a sensible default is used.
func generateQRCodePNG(content string, size int) ([]byte, error) {
	if size <= 0 {
		size = 256
	}
	// Use high-recovery level M for reasonable resilience
	q, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	// Encode to PNG with the requested size
	var buf bytes.Buffer
	if err := q.Write(size, &buf); err != nil {
		return nil, err
	}

	// Ensure valid PNG by decoding/encoding once (gofpdf expects proper PNG streams).
	// This step also allows us to control PNG encoding settings if needed.
	im, err := png.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		// If decode fails, return the raw bytes (qrcode lib should produce valid PNGs,
		// but we keep this fallback).
		return buf.Bytes(), nil
	}
	var out bytes.Buffer
	if err := png.Encode(&out, im); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// registerImageFromBytes registers an image with gofpdf using bytes.Readers.
// gofpdf requires a stable reader; we pass a new reader for the content.
func registerImageFromBytes(pdf *gofpdf.Fpdf, name string, opt gofpdf.ImageOptions, b []byte) error {
	// gofpdf's RegisterImageOptionsReader requires an io.Reader. Use a bytes.Reader.
	reader := bytes.NewReader(b)
	// The RegisterImageOptionsReader will read the supplied reader to get the image
	// information and store it under the provided name.
	pdf.RegisterImageOptionsReader(name, opt, reader)
	// After registration, the image can be referenced by the same name in ImageOptions.
	// Note: RegisterImageOptionsReader does not return an error, but it may panic on invalid images.
	return nil
}

// WritePDFToWriter is a small helper that creates a PDF for a single set of codes
// and writes the output to the given writer instead of a file. Useful for tests or
// streaming scenarios. It writes the bytes of the PDF to the provided io.Writer.
func WritePDFToWriter(w io.Writer, codes []string, arg *types.Argument) error {
	// Create an in-memory PDF using same layout as GeneratePDF.
	// We'll produce a temporary pdf and copy its bytes into the writer.
	tmpDir, err := os.MkdirTemp("", "qrgen-pdf")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	path, err := GeneratePDF(tmpDir, 0, codes, arg)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}
