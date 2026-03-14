package axon

import (
	"database/sql"
	"embed"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// MustOpenDB opens a PostgreSQL connection, creates the schema if needed,
// sets the search_path, and pings to verify connectivity.
// Panics on failure.
func MustOpenDB(dsn, schema string) *sql.DB {
	db, err := OpenDB(dsn, schema)
	if err != nil {
		panic(fmt.Sprintf("axon: open database (schema %s): %v", schema, err))
	}
	return db
}

// OpenDB opens a PostgreSQL connection, creates the schema if needed,
// and sets the search_path via the DSN so it applies to all pooled connections.
func OpenDB(dsn, schema string) (*sql.DB, error) {
	// First open a temporary connection to create the schema
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if _, err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + pgx.Identifier{schema}.Sanitize()); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema %s: %w", schema, err)
	}
	db.Close()

	// Reopen with search_path baked into the DSN so every pooled connection uses it
	dsnWithSchema := appendSearchPath(dsn, schema)
	db, err = sql.Open("pgx", dsnWithSchema)
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
func RunMigrations(db *sql.DB, migrationsFS embed.FS) error {
	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(log.New(io.Discard, "", 0))

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	slog.Info("database migrations complete")
	return nil
}

// MustRunMigrations runs goose SQL migrations. Panics on failure.
func MustRunMigrations(db *sql.DB, migrationsFS embed.FS) {
	if err := RunMigrations(db, migrationsFS); err != nil {
		panic(fmt.Sprintf("axon: run migrations: %v", err))
	}
}
