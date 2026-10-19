// Package access owns staff permissions and explicitly scoped organizational access.
package access

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/astawh/web/internal/secret"
)

var (
	ErrForbidden = errors.New("access denied")
	ErrInvalid   = errors.New("invalid input")
	ErrConflict  = errors.New("operation conflicts with current state")
)

const (
	ManageUsers         = "users.manage"
	ManageOrganization  = "organization.manage"
	ManageContent       = "content.manage"
	ProcessApplications = "applications.process"
)

type Querier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func Require(ctx context.Context, q Querier, user, permission string) error {
	var allowed bool
	err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users u
 JOIN user_roles ur ON ur.user_id=u.id JOIN role_permissions rp ON rp.role_id=ur.role_id
 WHERE u.id=? AND u.disabled_at IS NULL AND rp.permission_id=?)`, user, permission).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

type Membership struct {
	UnitID string `json:"unit_id"`
	Access string `json:"access"`
}
type Grants struct {
	Roles       []string     `json:"roles"`
	Memberships []Membership `json:"memberships"`
}
type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Disabled bool   `json:"disabled"`
}
type Unit struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

// ValidateGrants accepts only seeded roles and existing, explicit units.
func ValidateGrants(ctx context.Context, tx *sql.Tx, g Grants) error {
	seen := map[string]bool{}
	for _, role := range g.Roles {
		var found bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM roles WHERE id=?)`, role).Scan(&found); err != nil {
			return err
		}
		if !found || seen["role:"+role] {
			return ErrInvalid
		}
		seen["role:"+role] = true
	}
	for _, m := range g.Memberships {
		var found bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM organization_units WHERE id=?)`, m.UnitID).Scan(&found); err != nil {
			return err
		}
		if !found || seen["unit:"+m.UnitID] || (m.Access != "view" && m.Access != "manage") {
			return ErrInvalid
		}
		seen["unit:"+m.UnitID] = true
	}
	return nil
}

func Audit(ctx context.Context, tx *sql.Tx, actor, subject, action string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO access_events(actor_id,subject_id,action,created_at) VALUES(NULLIF(?,''),?,?,?)`, actor, subject, action, time.Now().Unix())
	return err
}

func (s *Service) CreateUnit(ctx context.Context, actor, name, kind string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 200 || (kind != "board" && kind != "department" && kind != "project") {
		return "", ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if err := Require(ctx, tx, actor, ManageOrganization); err != nil {
		return "", err
	}
	id := secret.New()
	if _, err := tx.ExecContext(ctx, `INSERT INTO organization_units VALUES(?,?,?,?)`, id, name, kind, time.Now().Unix()); err != nil {
		return "", err
	}
	if err := Audit(ctx, tx, actor, id, "unit.created"); err != nil {
		return "", err
	}
	return id, tx.Commit()
}

func (s *Service) Units(ctx context.Context, actor string) ([]Unit, error) {
	if err := Require(ctx, s.db, actor, ManageUsers); err != nil {
		if !errors.Is(err, ErrForbidden) {
			return nil, err
		}
		if err := Require(ctx, s.db, actor, ManageOrganization); err != nil {
			return nil, err
		}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,kind FROM organization_units ORDER BY name,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Unit{}
	for rows.Next() {
		var u Unit
		if err := rows.Scan(&u.ID, &u.Name, &u.Kind); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

func (s *Service) Users(ctx context.Context, actor string) ([]User, error) {
	if err := Require(ctx, s.db, actor, ManageUsers); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,email,disabled_at IS NOT NULL FROM users ORDER BY name,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Disabled); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

func (s *Service) UserGrants(ctx context.Context, actor, user string) (Grants, error) {
	if err := Require(ctx, s.db, actor, ManageUsers); err != nil {
		return Grants{}, err
	}
	return ReadGrants(ctx, s.db, user)
}

func ReadGrants(ctx context.Context, db *sql.DB, user string) (Grants, error) {
	g := Grants{Roles: []string{}, Memberships: []Membership{}}
	rows, err := db.QueryContext(ctx, `SELECT role_id FROM user_roles WHERE user_id=? ORDER BY role_id`, user)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			rows.Close()
			return g, err
		}
		g.Roles = append(g.Roles, r)
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return g, err
	}
	rows, err = db.QueryContext(ctx, `SELECT organization_unit_id,access_level FROM user_memberships WHERE user_id=? ORDER BY organization_unit_id`, user)
	if err != nil {
		return g, err
	}
	defer rows.Close()
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.UnitID, &m.Access); err != nil {
			return g, err
		}
		g.Memberships = append(g.Memberships, m)
	}
	return g, rows.Err()
}

func (s *Service) SetUserAccess(ctx context.Context, actor, user string, grants Grants, disabled bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := Require(ctx, tx, actor, ManageUsers); err != nil {
		return err
	}
	if err := ValidateGrants(ctx, tx, grants); err != nil {
		return err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=?)`, user).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrInvalid
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_roles WHERE user_id=?`, user); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_memberships WHERE user_id=?`, user); err != nil {
		return err
	}
	for _, r := range grants.Roles {
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_roles VALUES(?,?)`, user, r); err != nil {
			return err
		}
	}
	for _, m := range grants.Memberships {
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_memberships VALUES(?,?,?)`, user, m.UnitID, m.Access); err != nil {
			return err
		}
	}
	var disabledAt any
	if disabled {
		disabledAt = time.Now().Unix()
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET disabled_at=?,updated_at=? WHERE id=?`, disabledAt, time.Now().Unix(), user); err != nil {
		return err
	}
	var admins int
	if err := tx.QueryRowContext(ctx, `SELECT count(DISTINCT u.id) FROM users u JOIN user_roles ur ON ur.user_id=u.id JOIN role_permissions rp ON rp.role_id=ur.role_id WHERE u.disabled_at IS NULL AND rp.permission_id='users.manage'`).Scan(&admins); err != nil {
		return err
	}
	if admins == 0 {
		return fmt.Errorf("%w: last access administrator must remain enabled", ErrConflict)
	}
	if disabled {
		if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=?`, user); err != nil {
			return err
		}
	}
	if err := Audit(ctx, tx, actor, user, "user.access_changed"); err != nil {
		return err
	}
	return tx.Commit()
}

// Form access requires both a functional permission and an explicit membership.
// No role, including administrator, bypasses the responsibility boundary.
func (s *Service) CanAccessForm(ctx context.Context, user, form string, write bool) error {
	var allowed bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM form_responsibilities f
 JOIN user_memberships m ON m.organization_unit_id=f.organization_unit_id
 JOIN forms ON forms.id=f.form_id JOIN users u ON u.id=m.user_id
 WHERE u.id=? AND u.disabled_at IS NULL AND f.form_id=? AND forms.active=1
 AND EXISTS(SELECT 1 FROM user_roles ur JOIN role_permissions rp ON rp.role_id=ur.role_id
 WHERE ur.user_id=u.id AND rp.permission_id='applications.process')
 AND (?=0 OR m.access_level='manage'))`, user, form, write).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (s *Service) CreateForm(ctx context.Context, actor, key, title string, units []string) (string, error) {
	key, title = strings.TrimSpace(key), strings.TrimSpace(title)
	if key == "" || title == "" || len(key) > 100 || len(title) > 200 {
		return "", ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if err := Require(ctx, tx, actor, ManageOrganization); err != nil {
		return "", err
	}
	g := Grants{}
	for _, unit := range units {
		g.Memberships = append(g.Memberships, Membership{UnitID: unit, Access: "view"})
	}
	if err := ValidateGrants(ctx, tx, g); err != nil {
		return "", err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM forms WHERE key=?)`, key).Scan(&exists); err != nil {
		return "", err
	}
	if exists {
		return "", ErrConflict
	}
	id := secret.New()
	if _, err := tx.ExecContext(ctx, `INSERT INTO forms(id,key,title) VALUES(?,?,?)`, id, key, title); err != nil {
		return "", err
	}
	for _, unit := range units {
		if _, err := tx.ExecContext(ctx, `INSERT INTO form_responsibilities VALUES(?,?)`, id, unit); err != nil {
			return "", err
		}
	}
	if err := Audit(ctx, tx, actor, id, "form.created"); err != nil {
		return "", err
	}
	return id, tx.Commit()
}
