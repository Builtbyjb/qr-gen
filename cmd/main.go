package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/yourusername/qrgen/pkg/parser"
	"github.com/yourusername/qrgen/pkg/service"
)

const version = "1.0.0"

const usageText = `Usage: qrgen [flags]

Required:
  --quantity=N       Number of QR codes to generate (must be > 0)
  --url=URL          Base URL embedded in each QR code (e.g. https://example.com)

Optional:
  --size=N           Canvas size in pixels (default: 50)
  --info=TEXT        Informational label attached to the run
  --format=pdf       Output format — only "pdf" is supported
  --storage=X        Storage backend: local (default), s3, gcs
  --chunk-size=N     Codes per PDF batch (default: 500)
  --project-id=X     GCS project ID (required for gcs storage)
  --bucket=X         Storage bucket name (required for s3/gcs storage)
  --send-email       Send the generated zip via Gmail when complete
  --email-to=X       Recipient email address (required with --send-email)
  --help, -h         Show this help message and exit
  --version, -v      Print version and exit

Environment variables:
  SECRET_KEY         Required — used to derive the hash embedded in each code
  GOOGLE_APPLICATION_CREDENTIALS  Path to GCS service-account JSON (for gcs storage)
  AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY  AWS credentials (for s3 storage)
  GOOGLE_OAUTH_CREDENTIALS        Path to OAuth client credentials (for --send-email)
  GMAIL_TOKEN_PATH                Path to stored Gmail OAuth token
  DATABASE_URL or PGHOST/PGUSER/PGPASSWORD/PGDATABASE  Postgres connection (optional)
  DEBUG=1            Enable verbose logging`

func main() {
	arg, err := parser.ParseArgs(os.Args[1:])
	if err != nil {
		switch err.Error() {
		case "help":
			fmt.Println(usageText)
			return
		case "version":
			fmt.Printf("qrgen %s\n", version)
			return
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		fmt.Fprintln(os.Stderr, "Run with --help for usage information.")
		os.Exit(1)
	}

	if os.Getenv("SECRET_KEY") == "" {
		fmt.Fprintln(os.Stderr, "error: environment variable SECRET_KEY is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	svc := service.New(arg)
	if err := svc.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "generation failed:", err)
		os.Exit(1)
	}

	fmt.Println("QR codes generated successfully")
}
