package access_test

import (
	"context"
	"errors"
	"testing"

	"github.com/astawh/web/internal/testdb"
	"github.com/astawh/web/model/access"
)

func TestExplicitOrganizationalAccess(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t)
	s := access.New(db)
	testdb.User(t, db, "admin", "administrator")
	testdb.User(t, db, "handler", "application_handler")
	testdb.User(t, db, "editor", "content_editor")
	finance, err := s.CreateUnit(ctx, "admin", "Finanzen", "department")
	if err != nil {
		t.Fatal(err)
	}
	board, err := s.CreateUnit(ctx, "admin", "Vorstand", "board")
	if err != nil {
		t.Fatal(err)
	}
	form, err := s.CreateForm(ctx, "admin", "loan", "Darlehen", []string{finance})
	if err != nil {
		t.Fatal(err)
	}
	g := access.Grants{Roles: []string{"application_handler"}, Memberships: []access.Membership{{UnitID: board, Access: "manage"}}}
	if err := s.SetUserAccess(ctx, "admin", "handler", g, false); err != nil {
		t.Fatal(err)
	}
	if err := s.CanAccessForm(ctx, "handler", form, false); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("board inherited finance access: %v", err)
	}
	g.Memberships = []access.Membership{{UnitID: finance, Access: "view"}}
	if err := s.SetUserAccess(ctx, "admin", "handler", g, false); err != nil {
		t.Fatal(err)
	}
	if err := s.CanAccessForm(ctx, "handler", form, false); err != nil {
		t.Fatal(err)
	}
	if err := s.CanAccessForm(ctx, "handler", form, true); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("view granted write: %v", err)
	}
	g.Memberships[0].Access = "manage"
	if err := s.SetUserAccess(ctx, "admin", "handler", g, false); err != nil {
		t.Fatal(err)
	}
	if err := s.CanAccessForm(ctx, "handler", form, true); err != nil {
		t.Fatal(err)
	}
	if err := s.SetFormResponsibilities(ctx, "admin", form, []string{board}, true); err != nil {
		t.Fatal(err)
	}
	if err := s.CanAccessForm(ctx, "handler", form, false); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("old form responsibility retained access: %v", err)
	}
	if err := s.SetFormResponsibilities(ctx, "admin", form, []string{finance}, false); err != nil {
		t.Fatal(err)
	}
	if err := s.CanAccessForm(ctx, "handler", form, false); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("inactive form accessible: %v", err)
	}
	if err := s.SetFormResponsibilities(ctx, "admin", form, []string{finance}, true); err != nil {
		t.Fatal(err)
	}
	if err := s.CanAccessForm(ctx, "admin", form, false); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("admin bypassed form access: %v", err)
	}
	if err := s.SetUserAccess(ctx, "editor", "editor", access.Grants{Roles: []string{"administrator"}}, false); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("self escalation: %v", err)
	}
	if err := s.SetUserAccess(ctx, "admin", "handler", g, true); err != nil {
		t.Fatal(err)
	}
	if err := s.CanAccessForm(ctx, "handler", form, false); !errors.Is(err, access.ErrForbidden) {
		t.Fatalf("disabled user retained access: %v", err)
	}
}

func TestLastAdministratorAndInvalidGrantsRollback(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t)
	s := access.New(db)
	testdb.User(t, db, "admin", "administrator")
	for _, g := range []access.Grants{{}, {Roles: []string{"unknown"}}, {Roles: []string{"administrator", "administrator"}}} {
		if err := s.SetUserAccess(ctx, "admin", "admin", g, false); err == nil {
			t.Fatal("invalid change accepted")
		}
		if err := access.Require(ctx, db, "admin", access.ManageUsers); err != nil {
			t.Fatalf("rollback lost administrator: %v", err)
		}
	}
	if err := s.SetUserAccess(ctx, "admin", "admin", access.Grants{Roles: []string{"administrator"}}, true); !errors.Is(err, access.ErrConflict) {
		t.Fatalf("disabled last administrator: %v", err)
	}
}
