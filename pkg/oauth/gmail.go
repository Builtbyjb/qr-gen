package oauth

// Gmail OAuth helper (stubbed, gated).
//
// This helper provides lightweight helpers to:
// - locate the Google OAuth client credentials JSON (from env var)
// - load / save OAuth2 tokens to a file
// - create an *http.Client using the stored token
//
// Behavior notes:
// - If no credentials file is found (env var GOOGLE_OAUTH_CREDENTIALS), GetClient
//   returns an error.
// - If a token file does not exist, GetClient returns an instructional error that
//   includes the authorization URL to visit to obtain an authorization code.
// - A convenience function ObtainTokenFromWeb is provided to exchange an auth code
//   read from stdin for a token and save it to disk. This is interactive and gated
//   by the presence of credentials and a writable token path.
//
// Environment variables:
// - GOOGLE_OAUTH_CREDENTIALS: path to OAuth client_credentials JSON (required to obtain a token)
// - GMAIL_TOKEN_PATH: optional path to store/read the OAuth token (defaults to $TMP/gmail_token.json)
//
// This file intentionally avoids performing any network calls automatically;
// it is safe to import and use in environments where credentials are not present.
// Integration tests that exercise Gmail sending should be gated and require
// explicit setup of credentials and token files.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Env names
const (
	EnvGoogleCredentials = "GOOGLE_OAUTH_CREDENTIALS"
	EnvGmailTokenPath    = "GMAIL_TOKEN_PATH"
)

// CredentialsFilePath returns the path to the Google OAuth client credentials JSON.
// Returns an error when the env var is not set.
func CredentialsFilePath() (string, error) {
	if p := os.Getenv(EnvGoogleCredentials); strings.TrimSpace(p) != "" {
		return p, nil
	}
	return "", fmt.Errorf("environment variable %s not set", EnvGoogleCredentials)
}

// TokenFilePath returns where the token will be read/written.
// If GMAIL_TOKEN_PATH is set, it is used; otherwise defaults to a temp file.
func TokenFilePath() string {
	if p := os.Getenv(EnvGmailTokenPath); strings.TrimSpace(p) != "" {
		return p
	}
	return filepath.Join(os.TempDir(), "gmail_token.json")
}

// tokenFromFile loads an oauth2.Token from disk.
func tokenFromFile(path string) (*oauth2.Token, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(b, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

// saveToken writes an oauth2.Token to path (creates parent directories if needed).
func saveToken(path string, token *oauth2.Token) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating token directory: %w", err)
	}
	b, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	// write with restrictive permissions
	if err := ioutil.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}
	return nil
}

// GetClient attempts to return an *http.Client authenticated with Gmail scopes.
// - scopes: the OAuth scopes required (e.g. []string{"https://www.googleapis.com/auth/gmail.send"})
//
// Behavior:
//   - If credentials JSON is missing, returns an error.
//   - If a token file exists, it's used.
//   - If no token file exists, returns an informational error containing an auth URL
//     that the user can visit to obtain an authorization code, and guidance to
//     call ObtainTokenFromWeb to exchange and save the token.
//
// This function keeps the logic small and explicit; it does not automatically
// open browsers or attempt interactive flows by itself.
func GetClient(ctx context.Context, scopes []string) (*http.Client, error) {
	credPath, err := CredentialsFilePath()
	if err != nil {
		return nil, err
	}

	credBytes, err := ioutil.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("reading credentials file %s: %w", credPath, err)
	}

	config, err := google.ConfigFromJSON(credBytes, scopes...)
	if err != nil {
		return nil, fmt.Errorf("parsing credentials JSON: %w", err)
	}

	tokenPath := TokenFilePath()
	// Try to load token from disk
	if tok, err := tokenFromFile(tokenPath); err == nil {
		return config.Client(ctx, tok), nil
	}

	// No token present — return instructional error with auth URL.
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	msg := fmt.Sprintf(
		"no OAuth token found at %s\n\nTo obtain a token:\n  1) Visit the following URL in your browser:\n\n%s\n\n  2) Grant access and copy the authorization code.\n  3) Run the helper to exchange and save the token (ObtainTokenFromWeb) or create the token file manually.\n\nAlternatively set %s to point to a valid token file containing an oauth2 token JSON.\n",
		tokenPath, authURL, EnvGmailTokenPath,
	)
	return nil, fmt.Errorf(msg)
}

// ObtainTokenFromWeb performs an interactive exchange: it prints an authorization URL
// and reads the authorization code from stdin to exchange for a token, then saves it
// to the configured token path. This is intended to be run manually by an operator.
//
// NOTE: This function will block waiting for stdin input. Use it in an interactive
// shell only.
func ObtainTokenFromWeb(ctx context.Context, scopes []string) (*oauth2.Token, error) {
	credPath, err := CredentialsFilePath()
	if err != nil {
		return nil, err
	}
	credBytes, err := ioutil.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("reading credentials file %s: %w", credPath, err)
	}
	config, err := google.ConfigFromJSON(credBytes, scopes...)
	if err != nil {
		return nil, fmt.Errorf("parsing credentials JSON: %w", err)
	}

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Fprintf(os.Stderr, "Open the following URL in your browser then enter the authorization code:\n\n%s\n\nAuthorization code: ", authURL)

	reader := bufio.NewReader(os.Stdin)
	code, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading authorization code: %w", err)
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("no code provided")
	}

	tok, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	tokenPath := TokenFilePath()
	if err := saveToken(tokenPath, tok); err != nil {
		return nil, fmt.Errorf("saving token: %w", err)
	}

	return tok, nil
}
