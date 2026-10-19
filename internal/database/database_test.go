package database_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/astawh/web/internal/database"
)

func TestMigrationsAndConstraints(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "data", "sqlite.db")
	db, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO user_roles VALUES('missing','administrator')`); err == nil {
		t.Fatal("foreign key not enforced")
	}
	if _, err := db.Exec(`INSERT INTO organization_units VALUES('x','Finance','invalid',0)`); err == nil {
		t.Fatal("invalid unit kind accepted")
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM roles`).Scan(&count); err != nil || count != 4 {
		t.Fatalf("roles: %d %v", count, err)
	}
	db.Close()
	db, err = database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.QueryRow(`SELECT count(*) FROM roles`).Scan(&count); err != nil || count != 4 {
		t.Fatalf("migration repeated: %d %v", count, err)
	}
}
