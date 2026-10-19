package team_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/astawh/web/internal/datetime"
	"github.com/astawh/web/internal/testdb"
	"github.com/astawh/web/model/access"
	"github.com/astawh/web/model/team"
)

func setup(t *testing.T) (context.Context, *sql.DB, *team.Service, string) {
	t.Helper()
	ctx := context.Background()
	db := testdb.Open(t)
	testdb.User(t, db, "admin", "administrator")
	testdb.User(t, db, "member", "")
	testdb.User(t, db, "editor", "organization_editor")
	unit, err := access.New(db).CreateUnit(ctx, "admin", "EDV", "department")
	if err != nil {
		t.Fatal(err)
	}
	s := team.New(db)
	position, err := s.SavePosition(ctx, "admin", team.Position{UnitID: unit, Title: "EDV", ContactEmail: "edv@asta-wh.de", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	return ctx, db, s, position
}

func weekly() team.OfficeHour {
	return team.OfficeHour{Kind: "weekly", Weekday: 5, StartTime: "11:30", EndTime: "12:30", Campus: "gelsenkirchen", Room: "B2.1.29", MeetingMode: "in_person", LectureOnly: true}
}

func TestTeamLifecycleAndPublication(t *testing.T) {
	ctx, db, s, position := setup(t)
	input := team.Assignment{PositionID: position, UserID: "member", Hours: []team.OfficeHour{weekly()}}
	id, err := s.SaveAssignment(ctx, "editor", input)
	if err != nil {
		t.Fatal(err)
	}
	input.ID = id
	members, err := s.PublicMembers(ctx)
	if err != nil || len(members) != 0 {
		t.Fatalf("unpublished member: %+v %v", members, err)
	}
	input.Published = true
	input.Hours = append(input.Hours, team.OfficeHour{Kind: "appointment", Campus: "recklinghausen", Room: "B7.1.305", MeetingMode: "hybrid", Notes: "In der vorlesungsfreien Zeit nach Vereinbarung"})
	if _, err := s.SaveAssignment(ctx, "editor", input); err != nil {
		t.Fatal(err)
	}
	members, err = s.PublicMembers(ctx)
	if err != nil || len(members) != 1 {
		t.Fatalf("published member: %+v %v", members, err)
	}
	m := members[0]
	if m.Name != "member" || m.Title != "EDV" || m.UnitName != "EDV" || m.ContactEmail != "edv@asta-wh.de" || len(m.Hours) != 2 || m.Hours[0].Timezone != team.Timezone || m.Hours[1].Kind != "appointment" {
		t.Fatalf("incorrect team data: %+v", m)
	}
	if input.Hours[0].Timezone != "" {
		t.Fatal("save mutated caller input")
	}
	// A public responsibility must not become an access role or membership.
	var grants int
	if err := db.QueryRow(`SELECT (SELECT count(*) FROM user_roles WHERE user_id='member')+(SELECT count(*) FROM user_memberships WHERE user_id='member')`).Scan(&grants); err != nil || grants != 0 {
		t.Fatalf("position granted access: %d %v", grants, err)
	}
	positions, err := s.Positions(ctx, "editor")
	if err != nil || len(positions) != 1 {
		t.Fatal(positions, err)
	}
	p := positions[0]
	p.Active = false
	if _, err := s.SavePosition(ctx, "admin", p); err != nil {
		t.Fatal(err)
	}
	members, err = s.PublicMembers(ctx)
	if err != nil || len(members) != 0 {
		t.Fatal("inactive position published", members, err)
	}
	p.Active = true
	if _, err := s.SavePosition(ctx, "admin", p); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE users SET disabled_at=? WHERE id='member'`, datetime.Now()); err != nil {
		t.Fatal(err)
	}
	members, err = s.PublicMembers(ctx)
	if err != nil || len(members) != 0 {
		t.Fatal("disabled user published", members, err)
	}
	if _, err := s.SaveAssignment(ctx, "admin", input); !errors.Is(err, access.ErrInvalid) {
		t.Fatal("disabled user assigned", err)
	}
	if err := s.RemoveAssignment(ctx, "editor", id); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM office_hours`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("orphan office hours: %d %v", count, err)
	}
	if err := s.RemoveAssignment(ctx, "admin", id); !errors.Is(err, access.ErrInvalid) {
		t.Fatal(err)
	}
}

func TestAuthorizationAndValidation(t *testing.T) {
	ctx, _, s, position := setup(t)
	a := team.Assignment{PositionID: position, UserID: "member", Hours: []team.OfficeHour{weekly()}}
	for _, actor := range []string{"", "member"} {
		if _, err := s.SaveAssignment(ctx, actor, a); !errors.Is(err, access.ErrForbidden) {
			t.Fatal("unauthorized assignment", err)
		}
		if _, err := s.Positions(ctx, actor); !errors.Is(err, access.ErrForbidden) {
			t.Fatal("unauthorized positions", err)
		}
		if _, err := s.Assignments(ctx, actor); !errors.Is(err, access.ErrForbidden) {
			t.Fatal("unauthorized assignments", err)
		}
		if err := s.RemoveAssignment(ctx, actor, "missing"); !errors.Is(err, access.ErrForbidden) {
			t.Fatal("unauthorized removal", err)
		}
		if _, err := s.SavePosition(ctx, actor, team.Position{Title: "Title"}); !errors.Is(err, access.ErrForbidden) {
			t.Fatal("unauthorized position", err)
		}
	}
	invalid := []team.OfficeHour{}
	for _, modify := range []func(*team.OfficeHour){
		func(h *team.OfficeHour) { h.Weekday = 0 },
		func(h *team.OfficeHour) { h.Weekday = 8 },
		func(h *team.OfficeHour) { h.StartTime = "9:30" },
		func(h *team.OfficeHour) { h.EndTime = "24:00" },
		func(h *team.OfficeHour) { h.EndTime = h.StartTime },
		func(h *team.OfficeHour) { h.Timezone = "UTC" },
		func(h *team.OfficeHour) { h.Campus = "unknown" },
		func(h *team.OfficeHour) { h.Campus = "" },
		func(h *team.OfficeHour) { h.MeetingMode = "unknown" },
		func(h *team.OfficeHour) { h.Kind = "appointment" },
		func(h *team.OfficeHour) { h.ScheduleURL = "/shop" },
	} {
		h := weekly()
		modify(&h)
		invalid = append(invalid, h)
	}
	for _, h := range invalid {
		a.Hours = []team.OfficeHour{h}
		if _, err := s.SaveAssignment(ctx, "admin", a); !errors.Is(err, access.ErrInvalid) {
			t.Fatalf("accepted %+v: %v", h, err)
		}
	}
	for _, link := range []string{"javascript:alert(1)", "//evil.example/path", "/\\evil.example", "https://user:password@example.org/", "https://", ""} {
		a.Hours = []team.OfficeHour{{Kind: "external", ScheduleURL: link, MeetingMode: "remote"}}
		if _, err := s.SaveAssignment(ctx, "admin", a); !errors.Is(err, access.ErrInvalid) {
			t.Fatalf("accepted schedule URL %q: %v", link, err)
		}
	}
	h := weekly()
	h.StartTime = "12:00"
	h.EndTime = "13:00"
	a.Hours = []team.OfficeHour{weekly(), h}
	if _, err := s.SaveAssignment(ctx, "admin", a); !errors.Is(err, access.ErrConflict) {
		t.Fatal("overlapping hours accepted", err)
	}
	h.StartTime = "12:30"
	a.Hours = []team.OfficeHour{weekly(), h}
	id, err := s.SaveAssignment(ctx, "admin", a)
	if err != nil {
		t.Fatal("adjacent intervals rejected", err)
	}
	if _, err := s.SaveAssignment(ctx, "admin", a); !errors.Is(err, access.ErrConflict) {
		t.Fatal("duplicate assignment accepted", err)
	}
	a.ID = id
	a.Hours = []team.OfficeHour{{Kind: "external", ScheduleURL: "/shop", Campus: "gelsenkirchen", MeetingMode: "in_person"}}
	if _, err := s.SaveAssignment(ctx, "admin", a); err != nil {
		t.Fatal(err)
	}
	records, err := s.Assignments(ctx, "admin")
	if err != nil || len(records) != 1 || len(records[0].Hours) != 1 || records[0].Hours[0].ScheduleURL != "/shop" {
		t.Fatalf("replacement failed: %+v %v", records, err)
	}
	a.UserID = "missing"
	if _, err := s.SaveAssignment(ctx, "admin", a); !errors.Is(err, access.ErrInvalid) {
		t.Fatal("missing user accepted", err)
	}
}

func TestAssignmentRollbackAndDatabaseConstraints(t *testing.T) {
	ctx, db, s, position := setup(t)
	a := team.Assignment{PositionID: position, UserID: "member", Hours: []team.OfficeHour{weekly()}}
	id, err := s.SaveAssignment(ctx, "admin", a)
	if err != nil {
		t.Fatal(err)
	}
	a.ID = id
	if _, err := db.Exec(`CREATE TRIGGER fail_team_audit BEFORE INSERT ON access_events BEGIN SELECT RAISE(ABORT,'audit unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	a.Published = true
	a.Hours = nil
	if _, err := s.SaveAssignment(ctx, "admin", a); err == nil {
		t.Fatal("save without audit succeeded")
	}
	records, err := s.Assignments(ctx, "admin")
	if err != nil || len(records) != 1 || records[0].Published || len(records[0].Hours) != 1 {
		t.Fatalf("partial save: %+v %v", records, err)
	}
	if err := s.RemoveAssignment(ctx, "admin", id); err == nil {
		t.Fatal("delete without audit succeeded")
	}
	for _, statement := range []string{
		`UPDATE office_hours SET timezone='UTC'`,
		`UPDATE office_hours SET start_time='9:00'`,
		`UPDATE office_hours SET start_time='29:00'`,
		`UPDATE office_hours SET weekday=NULL`,
		`UPDATE office_hours SET weekday=8`,
		`UPDATE office_hours SET end_time=start_time`,
		`UPDATE office_hours SET kind='appointment'`,
		`UPDATE office_hours SET campus='unknown'`,
		`UPDATE office_hours SET campus=''`,
		`UPDATE office_hours SET assignment_id='missing'`,
	} {
		if _, err := db.Exec(statement); err == nil {
			t.Fatalf("invalid direct write accepted: %s", statement)
		}
	}
}

func TestWeeklyHoursRemainLocalAcrossDST(t *testing.T) {
	ctx, _, s, position := setup(t)
	if _, err := s.SaveAssignment(ctx, "admin", team.Assignment{PositionID: position, UserID: "member", Published: true, Hours: []team.OfficeHour{weekly()}}); err != nil {
		t.Fatal(err)
	}
	members, err := s.PublicMembers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	h := members[0].Hours[0]
	zone, err := time.LoadLocation(h.Timezone)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ date, utc string }{
		{"2026-01-16", "2026-01-16T10:30:00Z"},
		{"2026-07-17", "2026-07-17T09:30:00Z"},
	} {
		local, err := time.ParseInLocation("2006-01-02 15:04", tc.date+" "+h.StartTime, zone)
		if err != nil {
			t.Fatal(err)
		}
		if int(local.Weekday()) != h.Weekday || datetime.Format(local) != tc.utc {
			t.Fatalf("incorrect local recurrence: %v", local)
		}
	}
}

func TestSharedPositionsAndPrivateEmailBoundary(t *testing.T) {
	ctx, db, s, position := setup(t)
	testdb.User(t, db, "second", "")
	for i, user := range []string{"member", "second"} {
		if _, err := s.SaveAssignment(ctx, "admin", team.Assignment{PositionID: position, UserID: user, Published: true, SortOrder: 1 - i}); err != nil {
			t.Fatal(err)
		}
	}
	positions, err := s.Positions(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	p := positions[0]
	p.ID = ""
	p.Title = "Standortkoordination"
	p.ContactEmail = ""
	other, err := s.SavePosition(ctx, "admin", p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAssignment(ctx, "admin", team.Assignment{PositionID: other, UserID: "member", Published: true}); err != nil {
		t.Fatal(err)
	}
	members, err := s.PublicMembers(ctx)
	if err != nil || len(members) != 3 {
		t.Fatalf("multiple responsibilities: %+v %v", members, err)
	}
	if members[0].Name != "second" || members[1].Name != "member" || members[2].ContactEmail != "" {
		t.Fatalf("ordering or private email fallback: %+v", members)
	}
}
