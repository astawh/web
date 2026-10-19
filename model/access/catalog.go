package access

import (
	"context"
)

type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type Form struct {
	ID     string   `json:"id"`
	Key    string   `json:"key"`
	Title  string   `json:"title"`
	Active bool     `json:"active"`
	Units  []string `json:"units"`
}

func (s *Service) Roles(ctx context.Context, actor string) ([]Role, error) {
	if err := Require(ctx, s.db, actor, ManageUsers); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT r.id,r.name,COALESCE(p.permission_id,'') FROM roles r LEFT JOIN role_permissions p ON p.role_id=r.id ORDER BY r.id,p.permission_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Role{}
	for rows.Next() {
		var id, name, permission string
		if err := rows.Scan(&id, &name, &permission); err != nil {
			return nil, err
		}
		if len(result) == 0 || result[len(result)-1].ID != id {
			result = append(result, Role{ID: id, Name: name, Permissions: []string{}})
		}
		if permission != "" {
			result[len(result)-1].Permissions = append(result[len(result)-1].Permissions, permission)
		}
	}
	return result, rows.Err()
}

func (s *Service) Forms(ctx context.Context, actor string) ([]Form, error) {
	if err := Require(ctx, s.db, actor, ManageOrganization); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT f.id,f.key,f.title,f.active,COALESCE(r.organization_unit_id,'') FROM forms f LEFT JOIN form_responsibilities r ON r.form_id=f.id ORDER BY f.key,r.organization_unit_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Form{}
	for rows.Next() {
		var f Form
		var unit string
		if err := rows.Scan(&f.ID, &f.Key, &f.Title, &f.Active, &unit); err != nil {
			return nil, err
		}
		if len(result) == 0 || result[len(result)-1].ID != f.ID {
			f.Units = []string{}
			result = append(result, f)
		}
		if unit != "" {
			result[len(result)-1].Units = append(result[len(result)-1].Units, unit)
		}
	}
	return result, rows.Err()
}

func (s *Service) SetFormResponsibilities(ctx context.Context, actor, form string, units []string, active bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := Require(ctx, tx, actor, ManageOrganization); err != nil {
		return err
	}
	g := Grants{}
	for _, id := range units {
		g.Memberships = append(g.Memberships, Membership{UnitID: id, Access: "view"})
	}
	if err := ValidateGrants(ctx, tx, g); err != nil {
		return err
	}
	r, err := tx.ExecContext(ctx, `UPDATE forms SET active=? WHERE id=?`, active, form)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrInvalid
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM form_responsibilities WHERE form_id=?`, form); err != nil {
		return err
	}
	for _, unit := range units {
		if _, err := tx.ExecContext(ctx, `INSERT INTO form_responsibilities VALUES(?,?)`, form, unit); err != nil {
			return err
		}
	}
	if err := Audit(ctx, tx, actor, form, "form.responsibilities_changed"); err != nil {
		return err
	}
	return tx.Commit()
}
