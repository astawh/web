// Package events owns event content, publication and calendar scheduling.
package events

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"time"
	_ "time/tzdata" // All-day boundaries must work in the minimal runtime image.
	"unicode/utf8"

	"github.com/astawh/web/internal/datetime"
	"github.com/astawh/web/internal/secret"
	"github.com/astawh/web/model/access"
)

var ErrNotFound = errors.New("event not found")

const CalendarTimezone = "Europe/Berlin"

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

// Details is plain text. It must be escaped by the future SSR renderer.
type Details struct {
	Title, Description, Campus, Location, Link, Status string
}

// Schedule kinds are unscheduled, timed and all_day. Timed input requires
// RFC3339 instants with explicit zones. All-day input uses Berlin calendar dates;
// EndDate is exclusive and defaults to the day after StartDate.
type Schedule struct {
	Kind               string
	StartsAt, EndsAt   string
	StartDate, EndDate string
}

type Draft struct {
	Details
	Schedule Schedule
}

type Event struct {
	ID string
	Details
	AllDay               bool
	StartsAt, EndsAt     *string
	CreatedAt, UpdatedAt string
}

type Page struct{ Limit, Offset int }

func page(p Page) (Page, error) {
	if p.Limit == 0 {
		p.Limit = 20
	}
	if p.Limit < 1 || p.Limit > 100 || p.Offset < 0 {
		return p, access.ErrInvalid
	}
	return p, nil
}

func validText(value string, max int) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) <= max && !strings.ContainsRune(value, 0)
}

func normalizeDetails(d Details) (Details, error) {
	d.Title = strings.TrimSpace(d.Title)
	d.Description = strings.TrimSpace(d.Description)
	d.Location = strings.TrimSpace(d.Location)
	d.Link = strings.TrimSpace(d.Link)
	if d.Status == "" {
		d.Status = "draft"
	}
	if d.Title == "" || !validText(d.Title, 200) || !validText(d.Description, 50000) || !validText(d.Location, 500) || !validText(d.Link, 2048) {
		return d, access.ErrInvalid
	}
	if d.Campus != "" && d.Campus != "gelsenkirchen" && d.Campus != "recklinghausen" && d.Campus != "bocholt" {
		return d, access.ErrInvalid
	}
	if d.Status != "draft" && d.Status != "published" && d.Status != "cancelled" {
		return d, access.ErrInvalid
	}
	if d.Link != "" {
		u, err := url.Parse(d.Link)
		if err != nil || u.User != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || strings.ContainsAny(d.Link, "\r\n\t\\") {
			return d, access.ErrInvalid
		}
	}
	return d, nil
}

func instant(value string) (string, error) {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || t.Year() < 1 {
		return "", access.ErrInvalid
	}
	canonical := datetime.Format(t)
	if len(canonical) != 20 {
		return "", access.ErrInvalid
	}
	return canonical, nil
}

func normalizeSchedule(s Schedule) (bool, *string, *string, error) {
	if s.Kind == "" {
		s.Kind = "unscheduled"
	}
	switch s.Kind {
	case "unscheduled":
		if s.StartsAt != "" || s.EndsAt != "" || s.StartDate != "" || s.EndDate != "" {
			return false, nil, nil, access.ErrInvalid
		}
		return false, nil, nil, nil
	case "timed":
		if s.StartDate != "" || s.EndDate != "" {
			return false, nil, nil, access.ErrInvalid
		}
		start, err := instant(s.StartsAt)
		if err != nil {
			return false, nil, nil, err
		}
		var end *string
		if s.EndsAt != "" {
			value, err := instant(s.EndsAt)
			if err != nil || value <= start {
				return false, nil, nil, access.ErrInvalid
			}
			end = &value
		}
		return false, &start, end, nil
	case "all_day":
		if s.StartsAt != "" || s.EndsAt != "" {
			return false, nil, nil, access.ErrInvalid
		}
		zone, err := time.LoadLocation(CalendarTimezone)
		if err != nil {
			return false, nil, nil, err
		}
		start, err := time.ParseInLocation(time.DateOnly, s.StartDate, zone)
		if err != nil || start.Year() < 1 || start.Format(time.DateOnly) != s.StartDate {
			return false, nil, nil, access.ErrInvalid
		}
		end := start.AddDate(0, 0, 1)
		if s.EndDate != "" {
			end, err = time.ParseInLocation(time.DateOnly, s.EndDate, zone)
			if err != nil || end.Format(time.DateOnly) != s.EndDate {
				return false, nil, nil, access.ErrInvalid
			}
		}
		a, b := datetime.Format(start), datetime.Format(end)
		if len(a) != 20 || len(b) != 20 || !end.After(start) {
			return false, nil, nil, access.ErrInvalid
		}
		return true, &a, &b, nil
	default:
		return false, nil, nil, access.ErrInvalid
	}
}

// Save creates when id is empty, otherwise updates an existing, undeleted event.
// Saving an event never publishes implicitly: an omitted status means draft.
func (s *Service) Save(ctx context.Context, actor, id string, input Draft) (string, error) {
	d, err := normalizeDetails(input.Details)
	if err != nil {
		return "", err
	}
	allDay, start, end, err := normalizeSchedule(input.Schedule)
	if err != nil {
		return "", err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if err := access.Require(ctx, tx, actor, access.ManageContent); err != nil {
		return "", err
	}
	now := datetime.Now()
	if id == "" {
		id = secret.New()
		_, err = tx.ExecContext(ctx, `INSERT INTO events(id,title,description,campus,location,link,status,all_day,starts_at,ends_at,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, id, d.Title, d.Description, d.Campus, d.Location, d.Link, d.Status, allDay, start, end, now, now)
	} else {
		var result sql.Result
		result, err = tx.ExecContext(ctx, `UPDATE events SET title=?,description=?,campus=?,location=?,link=?,status=?,all_day=?,starts_at=?,ends_at=?,updated_at=? WHERE id=? AND deleted_at IS NULL`, d.Title, d.Description, d.Campus, d.Location, d.Link, d.Status, allDay, start, end, now, id)
		if err == nil {
			err = changed(result)
		}
	}
	if err != nil {
		return "", err
	}
	if err := access.Audit(ctx, tx, actor, id, "event.saved"); err != nil {
		return "", err
	}
	return id, tx.Commit()
}

func changed(result sql.Result) error {
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, actor, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := access.Require(ctx, tx, actor, access.ManageContent); err != nil {
		return err
	}
	now := datetime.Now()
	result, err := tx.ExecContext(ctx, `UPDATE events SET deleted_at=?,updated_at=? WHERE id=? AND deleted_at IS NULL`, now, now, id)
	if err != nil {
		return err
	}
	if err := changed(result); err != nil {
		return err
	}
	if err := access.Audit(ctx, tx, actor, id, "event.deleted"); err != nil {
		return err
	}
	return tx.Commit()
}

const selection = `SELECT id,title,description,campus,location,link,status,all_day,starts_at,ends_at,created_at,updated_at FROM events WHERE deleted_at IS NULL`

func scan(row interface{ Scan(...any) error }) (Event, error) {
	var e Event
	err := row.Scan(&e.ID, &e.Title, &e.Description, &e.Campus, &e.Location, &e.Link, &e.Status, &e.AllDay, &e.StartsAt, &e.EndsAt, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return e, err
}

func (s *Service) Lookup(ctx context.Context, actor, id string) (Event, error) {
	if err := access.Require(ctx, s.db, actor, access.ManageContent); err != nil {
		return Event{}, err
	}
	return scan(s.db.QueryRowContext(ctx, selection+` AND id=?`, id))
}

// PublicLookup includes cancelled announcements so their cancellation remains
// visible, but conceals drafts and deleted records as not found.
func (s *Service) PublicLookup(ctx context.Context, id string) (Event, error) {
	return scan(s.db.QueryRowContext(ctx, selection+` AND status IN ('published','cancelled') AND id=?`, id))
}

func query(ctx context.Context, db *sql.DB, statement string, args ...any) ([]Event, error) {
	rows, err := db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Event{}
	for rows.Next() {
		e, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (s *Service) List(ctx context.Context, actor string, p Page) ([]Event, error) {
	if err := access.Require(ctx, s.db, actor, access.ManageContent); err != nil {
		return nil, err
	}
	p, err := page(p)
	if err != nil {
		return nil, err
	}
	return query(ctx, s.db, selection+` ORDER BY starts_at IS NULL,starts_at,id LIMIT ? OFFSET ?`, p.Limit, p.Offset)
}

// PublicList periods: all, upcoming, past. Ongoing events count as upcoming.
// Unscheduled announcements sort last and are never past. Cancelled events are
// visible in all/past, but are not promoted in upcoming lists.
func (s *Service) PublicList(ctx context.Context, now time.Time, period string, p Page) ([]Event, error) {
	p, err := page(p)
	if err != nil {
		return nil, err
	}
	statement := selection + ` AND status IN ('published','cancelled')`
	args := []any{}
	order := ` ORDER BY starts_at IS NULL,starts_at,id`
	switch period {
	case "", "all":
	case "upcoming":
		statement += ` AND status='published' AND (starts_at IS NULL OR ends_at>? OR (ends_at IS NULL AND starts_at>=?))`
		args = append(args, datetime.Format(now), datetime.Format(now))
	case "past":
		statement += ` AND (ends_at<=? OR (ends_at IS NULL AND starts_at<?))`
		args = append(args, datetime.Format(now), datetime.Format(now))
		order = ` ORDER BY starts_at DESC,id`
	default:
		return nil, access.ErrInvalid
	}
	args = append(args, p.Limit, p.Offset)
	return query(ctx, s.db, statement+order+` LIMIT ? OFFSET ?`, args...)
}
