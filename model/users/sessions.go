package users

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/astawh/web/internal/secret"
	"github.com/astawh/web/model/access"
	"golang.org/x/crypto/bcrypt"
)

const SessionLifetime = 12 * time.Hour

func (s *Service) Grants(ctx context.Context, user string) (access.Grants, error) {
	return access.ReadGrants(ctx, s.db, user)
}

var dummyPasswordHash = sync.OnceValue(func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret.New()), 12)
	if err != nil {
		panic(err)
	}
	return hash
})

func (s *Service) Login(ctx context.Context, email, password string) (User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) > 254 || len(password) > 72 {
		return User{}, "", ErrCredentials
	}
	var u User
	var hash string
	var disabled bool
	err := s.db.QueryRowContext(ctx, `SELECT id,name,email,password_hash,disabled_at IS NOT NULL FROM users WHERE email=?`, email).Scan(&u.ID, &u.Name, &u.Email, &hash, &disabled)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return User{}, "", err
	}
	comparison := []byte(hash)
	if err != nil {
		comparison = dummyPasswordHash()
	}
	valid := bcrypt.CompareHashAndPassword(comparison, []byte(password)) == nil
	if err != nil || !valid || disabled {
		return User{}, "", ErrCredentials
	}
	token, now := secret.New(), time.Now().Unix()
	r, err := s.db.ExecContext(ctx, `INSERT INTO sessions(token_hash,user_id,created_at,expires_at)
 SELECT ?,id,?,? FROM users WHERE id=? AND disabled_at IS NULL AND password_hash=?`, secret.Hash(token), now, now+int64(SessionLifetime/time.Second), u.ID, hash)
	if err != nil {
		return User{}, "", err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return User{}, "", err
	}
	if n != 1 {
		return User{}, "", ErrCredentials
	}
	return u, token, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	if len(token) != 43 {
		return User{}, ErrCredentials
	}
	var u User
	err := s.db.QueryRowContext(ctx, `SELECT u.id,u.name,u.email FROM users u JOIN sessions s ON s.user_id=u.id
 WHERE s.token_hash=? AND s.expires_at>? AND u.disabled_at IS NULL`, secret.Hash(token), time.Now().Unix()).Scan(&u.ID, &u.Name, &u.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrCredentials
	}
	return u, err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=?`, secret.Hash(token))
	return err
}

func (s *Service) Cleanup(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at<=?`, time.Now().Unix())
	return err
}
