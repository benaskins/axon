package axon

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

// MustOpenDB opens a PostgreSQL connection, creates the schema if needed,
// sets the search_path, and pings to verify connectivity.
// Exits the process on failure.
func MustOpenDB(dsn, schema string) *sql.DB {
	db, err := OpenDB(dsn, schema)
	if err != nil {
		slog.Error("failed to open database", "error", err, "schema", schema)
		os.Exit(1)
	}
	return db
}

// OpenDB opens a PostgreSQL connection, creates the schema if needed,
// and sets the search_path via the DSN so it applies to all pooled connections.
func OpenDB(dsn, schema string) (*sql.DB, error) {
	// First open a temporary connection to create the schema
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if _, err := db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema %s: %w", schema, err)
	}
	db.Close()

	// Reopen with search_path baked into the DSN so every pooled connection uses it
	dsnWithSchema := appendSearchPath(dsn, schema)
	db, err = sql.Open("postgres", dsnWithSchema)
	if err != nil {
		return nil, fmt.Errorf("open database with search_path: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

// appendSearchPath adds search_path to a PostgreSQL DSN via the options parameter.
// Works with both URI format (postgres://...) and key=value format.
func appendSearchPath(dsn, schema string) string {
	opt := fmt.Sprintf("-csearch_path=%s", schema)
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return dsn
		}
		q := u.Query()
		q.Set("options", opt)
		u.RawQuery = q.Encode()
		return u.String()
	}
	// key=value format
	return dsn + " options=" + opt
}

// RunMigrations runs goose SQL migrations from an embedded filesystem.
// The migrations FS should embed a "migrations" directory containing SQL files
// (e.g., //go:embed migrations/*.sql).
func RunMigrations(db *sql.DB, migrationsFS embed.FS) {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("failed to set goose dialect", "error", err)
		os.Exit(1)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("database migrations complete")
}
