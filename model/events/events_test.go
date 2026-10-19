package events_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/astawh/web/internal/datetime"
	"github.com/astawh/web/internal/testdb"
	"github.com/astawh/web/model/access"
	"github.com/astawh/web/model/events"
)

func setup(t *testing.T) (context.Context, *sql.DB, *events.Service) {
	t.Helper()
	db := testdb.Open(t)
	testdb.User(t, db, "editor", "content_editor")
	testdb.User(t, db, "admin", "administrator")
	testdb.User(t, db, "member", "")
	return context.Background(), db, events.New(db)
}

func draft() events.Draft {
	return events.Draft{
		Details:  events.Details{Title: "Campusabend", Description: "Gemeinsam auf dem Campus", Campus: "gelsenkirchen", Location: "AStA", Link: "https://example.org/event"},
		Schedule: events.Schedule{Kind: "timed", StartsAt: "2026-10-19T18:00:00+02:00", EndsAt: "2026-10-19T20:00:00+02:00"},
	}
}

func TestEventLifecycleAndAuthorization(t *testing.T) {
	ctx, db, s := setup(t)
	d := draft()
	for _, actor := range []string{"", "member", "admin"} {
		if _, err := s.Save(ctx, actor, "", d); !errors.Is(err, access.ErrForbidden) {
			t.Fatalf("unauthorized save as %q: %v", actor, err)
		}
		if _, err := s.List(ctx, actor, events.Page{}); !errors.Is(err, access.ErrForbidden) {
			t.Fatal(err)
		}
		if _, err := s.Lookup(ctx, actor, "missing"); !errors.Is(err, access.ErrForbidden) {
			t.Fatal(err)
		}
		if err := s.Delete(ctx, actor, "missing"); !errors.Is(err, access.ErrForbidden) {
			t.Fatal(err)
		}
	}
	id, err := s.Save(ctx, "editor", "", d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.PublicLookup(ctx, id); !errors.Is(err, events.ErrNotFound) {
		t.Fatal("draft is public", err)
	}
	e, err := s.Lookup(ctx, "editor", id)
	if err != nil {
		t.Fatal(err)
	}
	if e.Status != "draft" || *e.StartsAt != "2026-10-19T16:00:00Z" || *e.EndsAt != "2026-10-19T18:00:00Z" || e.AllDay {
		t.Fatalf("incorrect normalized record: %+v", e)
	}
	d.Status = "published"
	if _, err := s.Save(ctx, "editor", id, d); err != nil {
		t.Fatal(err)
	}
	published, err := s.PublicLookup(ctx, id)
	if err != nil || published.CreatedAt != e.CreatedAt {
		t.Fatal(published, err)
	}
	d.Status = "cancelled"
	if _, err := s.Save(ctx, "editor", id, d); err != nil {
		t.Fatal(err)
	}
	published, err = s.PublicLookup(ctx, id)
	if err != nil || published.Status != "cancelled" {
		t.Fatal(published, err)
	}
	if _, err := db.Exec(`UPDATE users SET disabled_at=? WHERE id='editor'`, datetime.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "editor", id); !errors.Is(err, access.ErrForbidden) {
		t.Fatal("disabled editor can delete", err)
	}
	if _, err := db.Exec(`UPDATE users SET disabled_at=NULL WHERE id='editor'`); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "editor", id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PublicLookup(ctx, id); !errors.Is(err, events.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := s.Lookup(ctx, "editor", id); !errors.Is(err, events.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := s.Save(ctx, "editor", id, d); !errors.Is(err, events.ErrNotFound) {
		t.Fatal("updated deleted event", err)
	}
	if err := s.Delete(ctx, "editor", id); !errors.Is(err, events.ErrNotFound) {
		t.Fatal(err)
	}
	list, err := s.List(ctx, "editor", events.Page{})
	if err != nil || len(list) != 0 {
		t.Fatal(list, err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM access_events WHERE subject_id=?`, id).Scan(&count); err != nil || count != 4 {
		t.Fatalf("missing audit trail: %d %v", count, err)
	}
}

func TestAllDayEventsAcrossDST(t *testing.T) {
	ctx, _, s := setup(t)
	for _, tc := range []struct {
		date, start, end string
		hours            int
	}{
		{"2026-03-29", "2026-03-28T23:00:00Z", "2026-03-29T22:00:00Z", 23},
		{"2026-10-25", "2026-10-24T22:00:00Z", "2026-10-25T23:00:00Z", 25},
	} {
		d := draft()
		d.Schedule = events.Schedule{Kind: "all_day", StartDate: tc.date}
		id, err := s.Save(ctx, "editor", "", d)
		if err != nil {
			t.Fatal(err)
		}
		e, err := s.Lookup(ctx, "editor", id)
		if err != nil {
			t.Fatal(err)
		}
		if !e.AllDay || *e.StartsAt != tc.start || *e.EndsAt != tc.end {
			t.Fatalf("incorrect all-day boundaries: %+v", e)
		}
		start, _ := time.Parse(time.RFC3339, *e.StartsAt)
		end, _ := time.Parse(time.RFC3339, *e.EndsAt)
		if end.Sub(start) != time.Duration(tc.hours)*time.Hour {
			t.Fatal("DST day assumed to be 24 hours")
		}
	}
	d := draft()
	d.Schedule = events.Schedule{Kind: "all_day", StartDate: "2026-07-01", EndDate: "2026-07-04"}
	id, err := s.Save(ctx, "editor", "", d)
	if err != nil {
		t.Fatal(err)
	}
	e, err := s.Lookup(ctx, "editor", id)
	if err != nil || *e.EndsAt != "2026-07-03T22:00:00Z" {
		t.Fatal(e, err)
	}
}

func TestPublicTimelineAndPagination(t *testing.T) {
	ctx, _, s := setup(t)
	save := func(status, start, end string) string {
		t.Helper()
		d := draft()
		d.Status = status
		d.Schedule = events.Schedule{}
		if start != "" {
			d.Schedule = events.Schedule{Kind: "timed", StartsAt: start, EndsAt: end}
		}
		id, err := s.Save(ctx, "editor", "", d)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	past := save("published", "2026-10-19T12:00:00Z", "2026-10-19T13:00:00Z")
	ongoing := save("published", "2026-10-19T13:00:00Z", "2026-10-19T15:00:00Z")
	point := save("published", "2026-10-19T14:00:00Z", "")
	future := save("published", "2026-10-20T14:00:00Z", "")
	unknown := save("published", "", "")
	save("draft", "2026-10-19T14:00:00Z", "")
	cancelled := save("cancelled", "2026-10-20T14:00:00Z", "")
	now, _ := time.Parse(time.RFC3339, "2026-10-19T16:00:00+02:00")
	result, err := s.PublicList(ctx, now, "upcoming", events.Page{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{ongoing, point, future, unknown}
	if len(result) != len(want) {
		t.Fatalf("upcoming: %+v", result)
	}
	for i, e := range result {
		if e.ID != want[i] {
			t.Fatalf("incorrect timeline order: %+v", result)
		}
	}
	result, err = s.PublicList(ctx, now, "past", events.Page{})
	if err != nil || len(result) != 1 || result[0].ID != past {
		t.Fatalf("past: %+v %v", result, err)
	}
	result, err = s.PublicList(ctx, now, "upcoming", events.Page{Limit: 1, Offset: 2})
	if err != nil || len(result) != 1 || result[0].ID != future {
		t.Fatal(result, err)
	}
	all, err := s.PublicList(ctx, now, "all", events.Page{})
	if err != nil || len(all) != 6 {
		t.Fatal(all, err)
	}
	if _, err := s.PublicLookup(ctx, cancelled); err != nil {
		t.Fatal(err)
	}
	// An interval is past exactly at its exclusive end boundary.
	boundary, _ := time.Parse(time.RFC3339, "2026-10-19T15:00:00Z")
	result, err = s.PublicList(ctx, boundary, "upcoming", events.Page{})
	if err != nil || len(result) != 2 {
		t.Fatal(result, err)
	}
	for _, p := range []events.Page{{Limit: -1}, {Limit: 101}, {Offset: -1}} {
		if _, err := s.PublicList(ctx, now, "all", p); !errors.Is(err, access.ErrInvalid) {
			t.Fatal("invalid pagination accepted", err)
		}
	}
	if _, err := s.PublicList(ctx, now, "invalid", events.Page{}); !errors.Is(err, access.ErrInvalid) {
		t.Fatal(err)
	}
}

func TestInvalidEventInput(t *testing.T) {
	ctx, _, s := setup(t)
	for _, schedule := range []events.Schedule{
		{Kind: "timed", StartsAt: "2026-10-19T14:00:00"},
		{Kind: "timed", StartsAt: "2026-02-30T14:00:00Z"},
		{Kind: "timed", StartsAt: "2026-10-19T14:00:00Z", EndsAt: "2026-10-19T13:00:00Z"},
		{Kind: "timed", StartsAt: "2026-10-19T14:00:00.1Z", EndsAt: "2026-10-19T14:00:00.9Z"},
		{Kind: "all_day", StartDate: "2026-02-30"},
		{Kind: "all_day", StartDate: "2026-10-19", EndDate: "2026-10-19"},
		{Kind: "all_day", StartDate: "2026-10-19", StartsAt: "2026-10-19T00:00:00Z"},
		{Kind: "unscheduled", EndsAt: "2026-10-19T14:00:00Z"},
		{Kind: "recurring"},
	} {
		d := draft()
		d.Schedule = schedule
		if _, err := s.Save(ctx, "editor", "", d); !errors.Is(err, access.ErrInvalid) {
			t.Fatalf("accepted %+v: %v", schedule, err)
		}
	}
	for _, modify := range []func(*events.Draft){
		func(d *events.Draft) { d.Title = " " },
		func(d *events.Draft) { d.Title = strings.Repeat("x", 201) },
		func(d *events.Draft) { d.Description = strings.Repeat("x", 50001) },
		func(d *events.Draft) { d.Status = "unknown" },
		func(d *events.Draft) { d.Campus = "unknown" },
		func(d *events.Draft) { d.Link = "javascript:alert(1)" },
		func(d *events.Draft) { d.Link = "https://user:pass@example.org" },
	} {
		d := draft()
		modify(&d)
		if _, err := s.Save(ctx, "editor", "", d); !errors.Is(err, access.ErrInvalid) {
			t.Fatal("invalid details accepted", err)
		}
	}
}

func TestEventTransactionsAndSQLConstraints(t *testing.T) {
	ctx, db, s := setup(t)
	d := draft()
	id, err := s.Save(ctx, "editor", "", d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_event_audit BEFORE INSERT ON access_events BEGIN SELECT RAISE(ABORT,'audit unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	d.Status = "published"
	if _, err := s.Save(ctx, "editor", id, d); err == nil {
		t.Fatal("unaudited update succeeded")
	}
	if _, err := s.Save(ctx, "editor", "", d); err == nil {
		t.Fatal("unaudited create succeeded")
	}
	if err := s.Delete(ctx, "editor", id); err == nil {
		t.Fatal("unaudited delete succeeded")
	}
	list, err := s.List(ctx, "editor", events.Page{})
	if err != nil || len(list) != 1 || list[0].Status != "draft" {
		t.Fatal("partial transaction", list, err)
	}
	for _, statement := range []string{
		`UPDATE events SET ends_at=starts_at`,
		`UPDATE events SET starts_at=NULL`,
		`UPDATE events SET status='unknown'`,
		`UPDATE events SET all_day=2`,
		`UPDATE events SET all_day=1,starts_at=NULL,ends_at=NULL`,
		`UPDATE events SET title=''`,
		`UPDATE events SET campus='unknown'`,
	} {
		if _, err := db.Exec(statement); err == nil {
			t.Fatalf("invalid SQL write accepted: %s", statement)
		}
	}
}

func TestTimedPrecisionAndUnknownDate(t *testing.T) {
	ctx, _, s := setup(t)
	d := draft()
	d.Schedule = events.Schedule{Kind: "timed", StartsAt: "2026-10-19T09:30:00.123-05:00"}
	id, err := s.Save(ctx, "editor", "", d)
	if err != nil {
		t.Fatal(err)
	}
	e, err := s.Lookup(ctx, "editor", id)
	if err != nil || *e.StartsAt != "2026-10-19T14:30:00Z" || e.EndsAt != nil {
		t.Fatal(e, err)
	}
	d.Schedule = events.Schedule{}
	if _, err := s.Save(ctx, "editor", id, d); err != nil {
		t.Fatal(err)
	}
	e, err = s.Lookup(ctx, "editor", id)
	if err != nil || e.StartsAt != nil || e.EndsAt != nil || e.AllDay {
		t.Fatal("cleared schedule retained values", e, err)
	}
	if _, err := s.Save(ctx, "editor", "missing", d); !errors.Is(err, events.ErrNotFound) {
		t.Fatal(err)
	}
}
