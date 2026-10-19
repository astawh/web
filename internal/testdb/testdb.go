// Package testdb creates isolated migrated databases for integration tests.
package testdb

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/astawh/web/internal/database"
	"github.com/astawh/web/internal/datetime"
)

func Open(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "sqlite.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func User(t *testing.T, db *sql.DB, id, role string) {
	t.Helper()
	now := datetime.Now()
	if _, err := db.Exec(`INSERT INTO users(id,name,email,password_hash,verified_at,created_at,updated_at) VALUES(?,?,?,'not-a-login-hash',?,?,?)`, id, id, id+"@example.org", now, now, now); err != nil {
		t.Fatal(err)
	}
	if role != "" {
		if _, err := db.Exec(`INSERT INTO user_roles VALUES(?,?)`, id, role); err != nil {
			t.Fatal(err)
		}
	}
}
