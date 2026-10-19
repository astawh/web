package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/astawh/web/internal/secret"
	"github.com/astawh/web/internal/testdb"
	"github.com/astawh/web/model/access"
	"github.com/astawh/web/model/users"
)

func TestBootstrapLoginAndSessionRevocation(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t)
	s := users.New(db)
	u, err := s.Bootstrap(ctx, "Administrator", "admin@example.org", "a sufficiently long password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Bootstrap(ctx, "Second", "second@example.org", "a sufficiently long password"); !errors.Is(err, access.ErrConflict) {
		t.Fatal("bootstrap remained open")
	}
	for _, email := range []string{"admin@example.org", "missing@example.org"} {
		if _, _, err := s.Login(ctx, email, "wrong password"); !errors.Is(err, users.ErrCredentials) {
			t.Fatalf("invalid login: %v", err)
		}
	}
	logged, token, err := s.Login(ctx, " ADMIN@example.org ", "a sufficiently long password")
	if err != nil || logged.ID != u.ID {
		t.Fatalf("login: %+v %v", logged, err)
	}
	var stored string
	if err := db.QueryRow(`SELECT token_hash FROM sessions`).Scan(&stored); err != nil || stored == token || stored != secret.Hash(token) {
		t.Fatal("raw session stored")
	}
	if authenticated, err := s.Authenticate(ctx, token); err != nil || authenticated.ID != u.ID {
		t.Fatalf("session: %+v %v", authenticated, err)
	}
	if err := s.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, users.ErrCredentials) {
		t.Fatal("logged-out session valid")
	}
	_, token, err = s.Login(ctx, u.Email, "a sufficiently long password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE sessions SET expires_at='1970-01-01T00:00:01Z',created_at='1970-01-01T00:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, users.ErrCredentials) {
		t.Fatal("expired session valid")
	}
	_, token, err = s.Login(ctx, u.Email, "a sufficiently long password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE users SET disabled_at='2026-01-01T00:00:00Z' WHERE id=?`, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, users.ErrCredentials) {
		t.Fatal("disabled user's session valid")
	}
	if _, _, err := s.Login(ctx, u.Email, "a sufficiently long password"); !errors.Is(err, users.ErrCredentials) {
		t.Fatal("disabled user signed in")
	}
}
