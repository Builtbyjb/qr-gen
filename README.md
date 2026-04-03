# QR Gen — Go Rewrite

This directory contains a Go rewrite of the original QR code generator. The goal of this port is to provide a minimal, idiomatic Go implementation that reproduces the core CLI behavior from the original project: generate unique QR codes, write a CSV of generated codes, and produce simple artifacts under `tmp/` for downstream processing (PDF, ZIP, etc.).

This README explains how the Go project is laid out, how to build and run it, and how to run the tests.

## Requirements

- Go 1.18+ (module-enabled)
- Optional: A POSIX-like shell for examples (macOS / Linux / WSL on Windows)

## Quick start

From the `qr-gen/golang` directory:

- Run the CLI directly:
  - `go run ./cmd/qrgen --quantity=10 --url=https://example.com --size=500 --format=pdf --storage=local`

- Build a binary:
  - `go build -o bin/qrgen ./cmd/qrgen`
  - `./bin/qrgen --quantity=10 --url=https://example.com`

- Run tests:
  - `go test ./...`

## Environment variables

- `SECRET_KEY` (required by the codegen package and tests)
  - Used to derive a short hash that is embedded into generated codes.
  - Example: `export SECRET_KEY="my-secret-value"`

The original Java project included database and cloud-related configs (Postgres, Google OAuth, cloud storage, etc.). In this Go rewrite the focus is on the core generation logic and local artifact creation, so database/cloud integration is omitted. If you need those features, they can be implemented later.

## Usage

Basic usage:

- Required:
  - `--quantity` — number of QR codes to generate (must be > 0)
  - `--url` — base URL that will be encoded with each code (e.g. `https://example.com`)
- Optional:
  - `--size` — canvas size in pixels (default: `50` in the current port; historical code used larger sizes for PDF rendering)
  - `--info` — informational string (kept for parity with the original CLI)
  - `--format` — output format (`pdf` is supported by parser validation)
  - `--storage` — storage target (defaults to `local`)

Example:

`go run ./cmd/qrgen --quantity=100 --url=https://example.com --size=500 --format=pdf --storage=local`

This will:
- Generate `quantity` codes (e.g. `QR-...`)
- Create a small CSV at `tmp/csv/qr_codes.csv` with one code per line
- (In the original project additional steps created PDFs, zipped folders, uploaded to cloud, and cleaned up. The current Go port creates CSVs and exposes the core functions to be extended.)

## Project layout

- `cmd/qrgen` — CLI entrypoint (`main.go`)
- `pkg/types` — shared types and argument struct
- `pkg/parser` — CLI argument parsing and utility `ParseTime` function
- `pkg/codegen` — code generation: base62 encoder, hash generation, QR code composition
- `pkg/service` — high-level orchestration: generate many codes and write CSV artifacts
- `tmp/` — runtime temporary artifacts (created at runtime):
  - `tmp/csv/` — CSV files with generated codes
  - `tmp/pdfs/` — (reserved; PDF generation not currently implemented in Go port)
  - `tmp/zips/` — (reserved)

Files of interest:
- `cmd/qrgen/main.go` — CLI program
- `pkg/parser/parser.go` — argument parsing and validations
- `pkg/codegen/codegen.go` — deterministic code creation logic
- `pkg/service/service.go` — creates CSV artifact and exposes entrypoint for generation

## Tests

Run the test suite:

```
go test ./...
```

A few important notes:
- Some unit tests require `SECRET_KEY` to be set. The test suite will set/unset environment variables where needed, but you can also `export SECRET_KEY` before running tests.
- The tests cover:
  - `ParseTime` formatting
  - CLI argument parsing
  - Base62 encoding and code uniqueness behavior

## Extending the port

The Go port intentionally focuses on the "generate codes and produce local artifacts" path. Typical extension points:

- PDF generation
  - Add a `pkg/pdf` package that renders PNGs / PDF pages (e.g. using `github.com/jung-kurt/gofpdf` or other maintained forks)
- Cloud storage upload
  - Add adapters under `pkg/storage` (S3, GCS, etc.). Keep an interface-driven design to allow swapping backends.
- Database integration
  - Add `pkg/db` with a small repository layer for storing generated codes (e.g. using `database/sql` or an ORM)
- Email sending (OAuth)
  - Add `pkg/email` with support for Gmail OAuth flows if you want automated delivery

## Contributing

- Follow idiomatic Go patterns:
  - Keep packages small and focused
  - Return errors instead of panicking (except in initialization where appropriate)
  - Add unit tests for new behavior
- Open a PR with a clear description and accompanying tests.

## License

The original project is MIT-licensed; check upstream for license details. This port follows the same license unless otherwise requested.

## Notes

- This Go port was created to be a concise, testable foundation. It intentionally omits heavy external dependencies (image/PDF processing, cloud SDKs, database drivers) to keep the codebase small and easy to reason about. If you want full feature parity with the original Java implementation, I can add progressive pieces (PDF generation, zipping, cloud upload, DB integration) upon request.
