package oauth

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailSender sends messages via the Gmail API. It requires an authenticated
// *http.Client which can be obtained via GetClient present in this package.
type GmailSender struct {
	service *gmailapi.Service
	from    string // typically "me"
}

// NewGmailSenderFromEnv attempts to construct a GmailSender using the environment
// credentials / token that other helpers in this package work with.
//
// It returns (nil, nil) when credentials/token are not present or initialization fails.
// Callers can decide whether to treat that as non-fatal.
func NewGmailSenderFromEnv(ctx context.Context) (*GmailSender, error) {
	// Try to obtain an authenticated http.Client using existing helpers.
	// This will produce a helpful error when credentials/token are not ready.
	client, err := GetClient(ctx, []string{gmailapi.GmailSendScope})
	if err != nil {
		// Credentials or token not present/ready: return nil without forcing failure.
		return nil, nil
	}

	// Create Gmail service with the authenticated client.
	srv, err := gmailapi.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	return &GmailSender{
		service: srv,
		from:    "me", // use authenticated account
	}, nil
}

// SendWithAttachment sends an email (plain text body) with an optional attachment.
// It builds a multipart/mixed MIME message, base64url-encodes it per Gmail API
// requirements and calls the Send endpoint.
//
// Parameters:
//   - to: recipient email address (e.g. "user@example.com")
//   - subject: email subject
//   - body: plain text body
//   - attachmentPath: local path to a file to attach (empty string => no attachment)
func (g *GmailSender) SendWithAttachment(ctx context.Context, to string, subject string, body string, attachmentPath string) error {
	if g == nil || g.service == nil {
		return fmt.Errorf("gmail sender not initialized")
	}
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("recipient 'to' is required")
	}

	raw, err := buildRawMessage(g.from, to, subject, body, attachmentPath)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	msg := &gmailapi.Message{
		Raw: raw,
	}

	// Use the API to send the message as the authorized user ("me").
	_, err = g.service.Users.Messages.Send("me", msg).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("gmail send error: %w", err)
	}
	return nil
}

// buildRawMessage builds a multipart MIME message and returns the base64url-encoded
// string expected by the Gmail API in the Message.Raw field.
func buildRawMessage(from, to, subject, body, attachmentPath string) (string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	boundary := mw.Boundary()

	// Write main headers
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n", boundary)
	fmt.Fprint(&buf, "\r\n") // end headers

	// Part 1: text/plain body
	textHeader := make(textproto.MIMEHeader)
	textHeader.Set("Content-Type", "text/plain; charset=utf-8")
	textHeader.Set("Content-Transfer-Encoding", "7bit")
	tw, err := mw.CreatePart(textHeader)
	if err != nil {
		return "", fmt.Errorf("create text part: %w", err)
	}
	if _, err := tw.Write([]byte(body)); err != nil {
		return "", fmt.Errorf("write text part: %w", err)
	}

	// Part 2: attachment (optional)
	if strings.TrimSpace(attachmentPath) != "" {
		// Read the file
		data, err := os.ReadFile(attachmentPath)
		if err != nil {
			return "", fmt.Errorf("read attachment %s: %w", attachmentPath, err)
		}

		filename := filepath.Base(attachmentPath)
		attachHeader := make(textproto.MIMEHeader)
		attachHeader.Set("Content-Type", "application/octet-stream; name=\""+filename+"\"")
		attachHeader.Set("Content-Transfer-Encoding", "base64")
		attachHeader.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

		aw, err := mw.CreatePart(attachHeader)
		if err != nil {
			return "", fmt.Errorf("create attachment part: %w", err)
		}

		// Encode attachment as standard base64 for the message body part.
		encoded := base64.StdEncoding.EncodeToString(data)
		// Write encoded data in 76-character lines per RFC recommendations.
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			if _, err := aw.Write([]byte(encoded[i:end] + "\r\n")); err != nil {
				return "", fmt.Errorf("write attachment part: %w", err)
			}
		}
	}

	// Close the multipart writer to finalize the MIME body.
	if err := mw.Close(); err != nil {
		return "", fmt.Errorf("finalize mime: %w", err)
	}

	// Gmail API expects the entire MIME message to be base64url-encoded (RFC 4648, URL-safe).
	raw := base64.URLEncoding.EncodeToString(buf.Bytes())
	// Trim any padding characters to be safe (Gmail tolerates both, but URL-safe w/o '=').
	raw = strings.TrimRight(raw, "=")

	return raw, nil
}
