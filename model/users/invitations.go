package users

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/astawh/web/internal/datetime"
	"github.com/astawh/web/internal/secret"
	"github.com/astawh/web/model/access"
)

const InvitationLifetime = 7 * 24 * time.Hour

type InvitationInput struct {
	Name   string        `json:"name"`
	Email  string        `json:"email"`
	Grants access.Grants `json:"grants"`
}

type Invitation struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	InvitedBy  string  `json:"invited_by"`
	CreatedAt  string  `json:"created_at"`
	ExpiresAt  string  `json:"expires_at"`
	AcceptedAt *string `json:"accepted_at"`
	RevokedAt  *string `json:"revoked_at"`
}

func (s *Service) Invite(ctx context.Context, actor string, input InvitationInput) (string, error) {
	name, email, err := identity(input.Name, input.Email)
	if err != nil {
		return "", err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if err := access.Require(ctx, tx, actor, access.ManageUsers); err != nil {
		return "", err
	}
	if err := access.ValidateGrants(ctx, tx, input.Grants); err != nil {
		return "", err
	}
	now := datetime.Now()
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email=?)`, email).Scan(&exists); err != nil {
		return "", err
	}
	if exists {
		return "", access.ErrConflict
	}
	// Retire expired invitations before enforcing one outstanding invitation per email.
	if _, err := tx.ExecContext(ctx, `DELETE FROM mail_outbox WHERE invitation_id IN (SELECT id FROM invitations WHERE email=? AND expires_at<=?)`, email, now); err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE invitations SET revoked_at=? WHERE email=? AND expires_at<=? AND revoked_at IS NULL AND accepted_at IS NULL`, now, email, now); err != nil {
		return "", err
	}
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM invitations WHERE email=? AND revoked_at IS NULL AND accepted_at IS NULL)`, email).Scan(&exists); err != nil {
		return "", err
	}
	if exists {
		return "", access.ErrConflict
	}
	id, token := secret.New(), secret.New()
	if _, err := tx.ExecContext(ctx, `INSERT INTO invitations(id,name,email,token_hash,invited_by,created_at,expires_at) VALUES(?,?,?,?,?,?,?)`, id, name, email, secret.Hash(token), actor, now, datetime.Format(time.Now().Add(InvitationLifetime))); err != nil {
		return "", err
	}
	for _, r := range input.Grants.Roles {
		if _, err := tx.ExecContext(ctx, `INSERT INTO invitation_roles VALUES(?,?)`, id, r); err != nil {
			return "", err
		}
	}
	for _, m := range input.Grants.Memberships {
		if _, err := tx.ExecContext(ctx, `INSERT INTO invitation_memberships VALUES(?,?,?)`, id, m.UnitID, m.Access); err != nil {
			return "", err
		}
	}
	if err := queueInvitation(ctx, tx, id, email, token, now); err != nil {
		return "", err
	}
	if err := access.Audit(ctx, tx, actor, id, "invitation.created"); err != nil {
		return "", err
	}
	return id, tx.Commit()
}

func queueInvitation(ctx context.Context, tx *sql.Tx, id, email, token string, now string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO mail_outbox(id,invitation_id,recipient,token,created_at,next_attempt_at)
 VALUES(?,?,?,?,?,?) ON CONFLICT(invitation_id) DO UPDATE SET token=excluded.token,
 attempts=0,next_attempt_at=excluded.next_attempt_at,lease_token=NULL,lease_until=NULL`, secret.New(), id, email, token, now, now)
	return err
}

func (s *Service) Invitations(ctx context.Context, actor string) ([]Invitation, error) {
	if err := access.Require(ctx, s.db, actor, access.ManageUsers); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,email,invited_by,created_at,expires_at,accepted_at,revoked_at FROM invitations ORDER BY created_at DESC,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Invitation{}
	for rows.Next() {
		var i Invitation
		if err := rows.Scan(&i.ID, &i.Name, &i.Email, &i.InvitedBy, &i.CreatedAt, &i.ExpiresAt, &i.AcceptedAt, &i.RevokedAt); err != nil {
			return nil, err
		}
		result = append(result, i)
	}
	return result, rows.Err()
}

func (s *Service) RevokeInvitation(ctx context.Context, actor, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := access.Require(ctx, tx, actor, access.ManageUsers); err != nil {
		return err
	}
	r, err := tx.ExecContext(ctx, `UPDATE invitations SET revoked_at=? WHERE id=? AND accepted_at IS NULL AND revoked_at IS NULL`, datetime.Now(), id)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrInvitation
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM mail_outbox WHERE invitation_id=?`, id); err != nil {
		return err
	}
	if err := access.Audit(ctx, tx, actor, id, "invitation.revoked"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ResendInvitation(ctx context.Context, actor, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := access.Require(ctx, tx, actor, access.ManageUsers); err != nil {
		return err
	}
	var email string
	err = tx.QueryRowContext(ctx, `SELECT email FROM invitations WHERE id=? AND accepted_at IS NULL AND revoked_at IS NULL`, id).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvitation
	}
	if err != nil {
		return err
	}
	token, now := secret.New(), datetime.Now()
	// The administrator resending the invitation takes responsibility for its grants.
	if _, err := tx.ExecContext(ctx, `UPDATE invitations SET token_hash=?,expires_at=?,invited_by=? WHERE id=?`, secret.Hash(token), datetime.Format(time.Now().Add(InvitationLifetime)), actor, id); err != nil {
		return err
	}
	if err := queueInvitation(ctx, tx, id, email, token, now); err != nil {
		return err
	}
	if err := access.Audit(ctx, tx, actor, id, "invitation.resent"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) AcceptInvitation(ctx context.Context, token, password string) (User, error) {
	if len(token) != 43 {
		return User{}, ErrInvitation
	}
	// Reject unavailable tokens before spending time hashing a password.
	var available bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM invitations WHERE token_hash=? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at>?)`, secret.Hash(token), datetime.Now()).Scan(&available); err != nil {
		return User{}, err
	}
	if !available {
		return User{}, ErrInvitation
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
	now := datetime.Now()
	u := User{ID: secret.New()}
	var id, actor string
	err = tx.QueryRowContext(ctx, `SELECT id,name,email,invited_by FROM invitations WHERE token_hash=? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at>?`, secret.Hash(token), now).Scan(&id, &u.Name, &u.Email, &actor)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrInvitation
	}
	if err != nil {
		return User{}, err
	}
	if err := access.Require(ctx, tx, actor, access.ManageUsers); err != nil {
		if errors.Is(err, access.ErrForbidden) {
			return User{}, ErrInvitation
		}
		return User{}, err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email=?)`, u.Email).Scan(&exists); err != nil {
		return User{}, err
	}
	if exists {
		return User{}, ErrInvitation
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO users(id,name,email,password_hash,verified_at,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, u.ID, u.Name, u.Email, hash, now, now, now); err != nil {
		return User{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_roles SELECT ?,role_id FROM invitation_roles WHERE invitation_id=?`, u.ID, id); err != nil {
		return User{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_memberships SELECT ?,organization_unit_id,access_level FROM invitation_memberships WHERE invitation_id=?`, u.ID, id); err != nil {
		return User{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE invitations SET accepted_at=? WHERE id=?`, now, id); err != nil {
		return User{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM mail_outbox WHERE invitation_id=?`, id); err != nil {
		return User{}, err
	}
	if err := access.Audit(ctx, tx, u.ID, id, "invitation.accepted"); err != nil {
		return User{}, err
	}
	return u, tx.Commit()
}
