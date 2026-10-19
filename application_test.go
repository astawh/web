package main

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/astawh/web/internal/config"
)

func TestApplicationMigrationAndShutdown(t *testing.T) {
	app, err := newApplication(context.Background(), config.Config{SiteName: "AStA WH", Database: filepath.Join(t.TempDir(), "sqlite.db"), PublicURL: "http://localhost:8090"})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	app.handler.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var count int
	if err := app.db.QueryRow(`SELECT count(*) FROM users`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("unexpected seeded account: %d %v", count, err)
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFirstAdministratorCommand(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "sqlite.db")
	password := filepath.Join(dir, "password")
	if err := os.WriteFile(password, []byte("a sufficiently long password\n"), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"-database", db, "-bootstrap-admin", "admin@example.org", "-bootstrap-password-file", password}
	if err := run(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), args); err == nil {
		t.Fatal("bootstrap remained available")
	}
}
