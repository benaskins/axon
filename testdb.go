package axon

import (
	"database/sql"
	"embed"
	"fmt"
	"math/rand"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// OpenTestDB creates a unique PostgreSQL schema for test isolation.
// It opens a connection, runs migrations, and registers cleanup to drop
// the schema when the test finishes.
func OpenTestDB(t *testing.T, dsn string, migrations embed.FS) *sql.DB {
	t.Helper()

	schema := fmt.Sprintf("test_%d_%d", time.Now().UnixNano(), rand.Int())

	db, err := OpenDB(dsn, schema)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	RunMigrations(db, migrations)

	t.Cleanup(func() {
		db.Close()
		cleanDB, err := sql.Open("postgres", dsn)
		if err == nil {
			cleanDB.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schema))
			cleanDB.Close()
		}
	})

	return db
}
