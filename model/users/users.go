// Package users manages invitation-only staff identities and authentication.
package users

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/astawh/web/internal/secret"
	"github.com/astawh/web/model/access"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrCredentials = errors.New("invalid credentials")
	ErrInvitation  = errors.New("invitation is invalid or unavailable")
)

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

func identity(name, email string) (string, string, error) {
	name, email = strings.TrimSpace(name), strings.ToLower(strings.TrimSpace(email))
	a, err := mail.ParseAddress(email)
	if err != nil || a.Address != email || len(email) > 254 || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 200 {
		return "", "", access.ErrInvalid
	}
	return name, email, nil
}

func passwordHash(password string) (string, error) {
	if utf8.RuneCountInString(password) < 12 || len(password) > 72 {
		return "", access.ErrInvalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(hash), err
}

// Bootstrap is available only while the database has no users. No account is
// seeded automatically and this method is not exposed over HTTP.
func (s *Service) Bootstrap(ctx context.Context, name, email, password string) (User, error) {
	name, email, err := identity(name, email)
	if err != nil {
		return User{}, err
	}
	hash, err := passwordHash(password)
	if err != nil {
		return User{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&count); err != nil {
		return User{}, err
	}
	if count != 0 {
		return User{}, access.ErrConflict
	}
	now := time.Now().Unix()
	u := User{ID: secret.New(), Name: name, Email: email}
	if _, err := tx.ExecContext(ctx, `INSERT INTO users(id,name,email,password_hash,verified_at,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, u.ID, name, email, hash, now, now, now); err != nil {
		return User{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_roles VALUES(?,'administrator')`, u.ID); err != nil {
		return User{}, err
	}
	if err := access.Audit(ctx, tx, u.ID, u.ID, "user.bootstrapped"); err != nil {
		return User{}, err
	}
	return u, tx.Commit()
}
