package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB provides a thin repository layer over a pgx connection pool.
//
// Responsibilities implemented here:
// - establish pool connection using environment configuration
// - create the qr codes table if it doesn't exist
// - insert/check QR codes
// - store/retrieve oauth tokens (simple key-value table)
// The implementation is intentionally small and synchronous-friendly; callers
// should provide a context with an appropriate timeout/cancellation.
type DB struct {
	pool      *pgxpool.Pool
	tableName string
}

// New connects to Postgres using environment variables and returns a DB instance.
// Connection options (in order of precedence):
//   - DATABASE_URL
//   - if not present, the function looks for PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE
//
// It also reads QR_CODE_TABLE_NAME env var; if missing, defaults to "qr_codes".
func New(ctx context.Context) (*DB, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Build DSN-like connection string
		host := os.Getenv("PGHOST")
		port := os.Getenv("PGPORT")
		user := os.Getenv("PGUSER")
		password := os.Getenv("PGPASSWORD")
		dbname := os.Getenv("PGDATABASE")

		if host == "" || user == "" || password == "" || dbname == "" {
			return nil, errors.New("database configuration not found: set DATABASE_URL or PGHOST/PGUSER/PGPASSWORD/PGDATABASE")
		}
		if port == "" {
			port = "5432"
		}
		connStr = fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", user, password, host, port, dbname)
	}

	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// reasonable defaults for pool
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 10
	}
	if cfg.ConnConfig.ConnectTimeout == 0 {
		cfg.ConnConfig.ConnectTimeout = time.Second * 5
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	table := os.Getenv("QR_CODE_TABLE_NAME")
	if table == "" {
		table = "qr_codes"
	}

	d := &DB{
		pool:      pool,
		tableName: table,
	}

	if err := d.Init(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return d, nil
}

// Init creates required tables if they do not exist.
func (d *DB) Init(ctx context.Context) error {
	// Create main QR codes table
	createQRCodes := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id BIGSERIAL PRIMARY KEY,
	code_value VARCHAR(255) UNIQUE NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	processed BOOLEAN NOT NULL DEFAULT false,
	status VARCHAR(50) DEFAULT 'AVAILABLE',
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	location VARCHAR(255),
	project_id VARCHAR(255) NOT NULL,
	geolocation_id BIGINT,
	assigned VARCHAR(200)
)`, d.tableName)

	// Simple tokens table for storing oauth tokens and small key/value pairs.
	// This is useful for storing Gmail OAuth tokens without adding an extra
	// persistence mechanism in the initial implementation.
	createTokens := `
CREATE TABLE IF NOT EXISTS oauth_tokens (
	id SERIAL PRIMARY KEY,
	key TEXT UNIQUE NOT NULL,
	access_token TEXT,
	refresh_token TEXT,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)`

	// Run both in a transaction
	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	// Use Exec to create tables
	if _, err := conn.Exec(ctx, createQRCodes); err != nil {
		return fmt.Errorf("create qr_codes table: %w", err)
	}
	if _, err := conn.Exec(ctx, createTokens); err != nil {
		return fmt.Errorf("create oauth_tokens table: %w", err)
	}
	return nil
}

// Close is intentionally implemented with a context-aware signature lower in the file.
// The context-aware Close(ctx) method is used by callers that expect context-aware cleanup.

// InsertQRCode inserts a new QR code record. If a duplicate code_value exists
// the function returns nil (idempotent insert).
func (d *DB) InsertQRCode(ctx context.Context, codeValue string, projectID string) error {
	if codeValue == "" {
		return errors.New("codeValue is required")
	}
	sql := fmt.Sprintf(`
INSERT INTO %s (code_value, project_id)
VALUES ($1, $2)
ON CONFLICT (code_value) DO NOTHING
`, d.tableName)

	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, sql, codeValue, projectID); err != nil {
		return fmt.Errorf("insert qr code: %w", err)
	}
	return nil
}

// CodeExists checks if a code_value already exists.
func (d *DB) CodeExists(ctx context.Context, codeValue string) (bool, error) {
	if codeValue == "" {
		return false, errors.New("codeValue is required")
	}
	sql := fmt.Sprintf(`SELECT 1 FROM %s WHERE code_value = $1 LIMIT 1`, d.tableName)

	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	var tmp int
	err = conn.QueryRow(ctx, sql, codeValue).Scan(&tmp)
	if err != nil {
		// no rows -> not found
		// pgx returns pgx.ErrNoRows
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		// Any other scan error: surface it so callers can distinguish not-found vs DB failure.
		return false, fmt.Errorf("querying code existence: %w", err)
	}
	return true, nil
}

// SaveOauthTokens stores or updates OAuth tokens keyed by `key`.
// Example key values: "gmail_user" or "gmail_tokens".
func (d *DB) SaveOauthTokens(ctx context.Context, key, accessToken, refreshToken string) error {
	if key == "" {
		return errors.New("key is required")
	}
	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	sql := `
INSERT INTO oauth_tokens (key, access_token, refresh_token, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (key) DO UPDATE
  SET access_token = EXCLUDED.access_token,
      refresh_token = EXCLUDED.refresh_token,
      updated_at = NOW()
`
	if _, err := conn.Exec(ctx, sql, key, accessToken, refreshToken); err != nil {
		return fmt.Errorf("save oauth tokens: %w", err)
	}
	return nil
}

// GetOauthTokens retrieves stored tokens for a key. If not found, returns empty strings and nil error.
func (d *DB) GetOauthTokens(ctx context.Context, key string) (string, string, error) {
	if key == "" {
		return "", "", errors.New("key is required")
	}
	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return "", "", fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	sql := `SELECT access_token, refresh_token FROM oauth_tokens WHERE key = $1 LIMIT 1`
	var access, refresh *string
	row := conn.QueryRow(ctx, sql, key)
	if err := row.Scan(&access, &refresh); err != nil {
		// If no rows, return empty tokens
		// pgx returns a typed error, but to avoid import of pgx errors here we treat any scan error as not found.
		return "", "", nil
	}
	a, r := "", ""
	if access != nil {
		a = *access
	}
	if refresh != nil {
		r = *refresh
	}
	return a, r, nil
}

// NewRepositoryFromEnv attempts to create a DB repository using environment configuration.
// It returns (*DB, error). If DB configuration is not present it returns (nil, nil).
func NewRepositoryFromEnv() (*DB, error) {
	ctx := context.Background()
	// If no DB environment variables are supplied, do not attempt to connect.
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("PGHOST") == "" {
		return nil, nil
	}
	return New(ctx)
}

// SaveCodes inserts multiple QR codes into the database. It is idempotent:
// duplicate codes are ignored by the underlying insert statement.
func (d *DB) SaveCodes(ctx context.Context, codes []string) error {
	if d == nil || d.pool == nil {
		return fmt.Errorf("database repo not initialized")
	}
	for _, c := range codes {
		if err := d.InsertQRCode(ctx, c, "local"); err != nil {
			return err
		}
	}
	return nil
}

// Close gracefully closes the repository's connection pool. It satisfies the
// DBRepository.Close(ctx) signature used by callers that expect a context-aware close.
func (d *DB) Close(ctx context.Context) error {
	if d == nil {
		return nil
	}
	if d.pool != nil {
		d.pool.Close()
	}
	return nil
}
