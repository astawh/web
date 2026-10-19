// Package team manages organizational positions and their public office hours.
// Positions and assignments never grant website permissions or memberships.
package team

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/astawh/web/internal/datetime"
	"github.com/astawh/web/internal/secret"
	"github.com/astawh/web/model/access"
)

const Timezone = "Europe/Berlin"

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

type Position struct {
	ID, UnitID, Title, ContactEmail string
	SortOrder                       int
	Active                          bool
}

// OfficeHour is either a weekly local-time interval, an appointment-only
// arrangement, or a link to an external schedule. Weekday uses ISO 1..7.
// Notes can describe e.g. Teams or appointments during the lecture-free period.
type OfficeHour struct {
	Kind                                   string
	Weekday                                int
	StartTime, EndTime                     string
	Timezone                               string
	Campus, Room, MeetingMode, ScheduleURL string
	LectureOnly                            bool
	Notes                                  string
}

type Assignment struct {
	ID, PositionID, UserID string
	Published              bool
	SortOrder              int
	Hours                  []OfficeHour
}

// Member contains explicitly published team information, never private email,
// credentials, access grants, or a separate public profile.
type Member struct {
	AssignmentID, PositionID, Name, Title, UnitID, UnitName, ContactEmail string
	Hours                                                                 []OfficeHour
}

func text(value string, max int) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) <= max && !strings.ContainsRune(value, 0)
}

func normalizePosition(p Position) (Position, error) {
	p.Title = strings.TrimSpace(p.Title)
	p.ContactEmail = strings.TrimSpace(p.ContactEmail)
	if p.Title == "" || !text(p.Title, 200) || !text(p.ContactEmail, 254) {
		return p, access.ErrInvalid
	}
	if p.ContactEmail != "" {
		address, err := mail.ParseAddress(p.ContactEmail)
		if err != nil || address.Address != p.ContactEmail || strings.ContainsAny(p.ContactEmail, "\r\n") {
			return p, access.ErrInvalid
		}
	}
	return p, nil
}

func validURL(value string) bool {
	if strings.ContainsAny(value, "\r\n\t\\") || strings.TrimSpace(value) != value {
		return false
	}
	u, err := url.Parse(value)
	if err != nil || u.User != nil {
		return false
	}
	return (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" ||
		u.Scheme == "" && u.Host == "" && strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//")
}

func normalizeHours(input []OfficeHour) ([]OfficeHour, error) {
	if len(input) > 50 {
		return nil, access.ErrInvalid
	}
	hours := append([]OfficeHour{}, input...)
	for i := range hours {
		h := &hours[i]
		h.Room = strings.TrimSpace(h.Room)
		h.Notes = strings.TrimSpace(h.Notes)
		if h.Timezone == "" {
			h.Timezone = Timezone
		}
		if h.Timezone != Timezone || !text(h.Room, 100) || !text(h.Notes, 2000) || !text(h.ScheduleURL, 2048) {
			return nil, access.ErrInvalid
		}
		if h.Campus != "" && h.Campus != "gelsenkirchen" && h.Campus != "recklinghausen" && h.Campus != "bocholt" {
			return nil, access.ErrInvalid
		}
		if h.MeetingMode != "in_person" && h.MeetingMode != "remote" && h.MeetingMode != "hybrid" {
			return nil, access.ErrInvalid
		}
		if h.MeetingMode != "remote" && h.Campus == "" {
			return nil, access.ErrInvalid
		}
		if h.Kind == "weekly" {
			start, err1 := time.Parse("15:04", h.StartTime)
			end, err2 := time.Parse("15:04", h.EndTime)
			if h.Weekday < 1 || h.Weekday > 7 || err1 != nil || err2 != nil || start.Format("15:04") != h.StartTime || end.Format("15:04") != h.EndTime || !start.Before(end) {
				return nil, access.ErrInvalid
			}
			for _, previous := range hours[:i] {
				if previous.Kind == "weekly" && previous.Weekday == h.Weekday && previous.StartTime < h.EndTime && h.StartTime < previous.EndTime {
					return nil, access.ErrConflict
				}
			}
		} else if (h.Kind != "appointment" && h.Kind != "external") || h.Weekday != 0 || h.StartTime != "" || h.EndTime != "" {
			return nil, access.ErrInvalid
		}
		if h.Kind == "external" {
			if !validURL(h.ScheduleURL) {
				return nil, access.ErrInvalid
			}
		} else if h.ScheduleURL != "" {
			return nil, access.ErrInvalid
		}
	}
	return hours, nil
}

// SavePosition creates when ID is empty; otherwise it updates an existing position.
func (s *Service) SavePosition(ctx context.Context, actor string, input Position) (string, error) {
	p, err := normalizePosition(input)
	if err != nil {
		return "", err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if err := access.Require(ctx, tx, actor, access.ManageOrganization); err != nil {
		return "", err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM organization_units WHERE id=?)`, p.UnitID).Scan(&exists); err != nil {
		return "", err
	}
	if !exists {
		return "", access.ErrInvalid
	}
	now := datetime.Now()
	if p.ID == "" {
		p.ID = secret.New()
		_, err = tx.ExecContext(ctx, `INSERT INTO team_positions(id,organization_unit_id,title,contact_email,sort_order,active,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, p.ID, p.UnitID, p.Title, p.ContactEmail, p.SortOrder, p.Active, now, now)
	} else {
		var result sql.Result
		result, err = tx.ExecContext(ctx, `UPDATE team_positions SET organization_unit_id=?,title=?,contact_email=?,sort_order=?,active=?,updated_at=? WHERE id=?`, p.UnitID, p.Title, p.ContactEmail, p.SortOrder, p.Active, now, p.ID)
		if err == nil {
			err = changed(result)
		}
	}
	if err != nil {
		return "", err
	}
	if err := access.Audit(ctx, tx, actor, p.ID, "team.position_saved"); err != nil {
		return "", err
	}
	return p.ID, tx.Commit()
}

func changed(result sql.Result) error {
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return access.ErrInvalid
	}
	return nil
}

// SaveAssignment replaces the assignment's complete office-hour arrangement
// atomically. Publishing is explicit; assigning a position grants no access.
func (s *Service) SaveAssignment(ctx context.Context, actor string, a Assignment) (string, error) {
	hours, err := normalizeHours(a.Hours)
	if err != nil {
		return "", err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if err := access.Require(ctx, tx, actor, access.ManageOrganization); err != nil {
		return "", err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users u,team_positions p WHERE u.id=? AND u.disabled_at IS NULL AND p.id=?)`, a.UserID, a.PositionID).Scan(&exists); err != nil {
		return "", err
	}
	if !exists {
		return "", access.ErrInvalid
	}
	var duplicate bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM team_assignments WHERE user_id=? AND position_id=? AND id!=?)`, a.UserID, a.PositionID, a.ID).Scan(&duplicate); err != nil {
		return "", err
	}
	if duplicate {
		return "", access.ErrConflict
	}
	now := datetime.Now()
	if a.ID == "" {
		a.ID = secret.New()
		_, err = tx.ExecContext(ctx, `INSERT INTO team_assignments(id,position_id,user_id,published,sort_order,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, a.ID, a.PositionID, a.UserID, a.Published, a.SortOrder, now, now)
	} else {
		var result sql.Result
		result, err = tx.ExecContext(ctx, `UPDATE team_assignments SET position_id=?,user_id=?,published=?,sort_order=?,updated_at=? WHERE id=?`, a.PositionID, a.UserID, a.Published, a.SortOrder, now, a.ID)
		if err == nil {
			err = changed(result)
		}
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM office_hours WHERE assignment_id=?`, a.ID); err != nil {
		return "", err
	}
	for i, h := range hours {
		var weekday, start, end any
		if h.Kind == "weekly" {
			weekday = h.Weekday
			start = h.StartTime
			end = h.EndTime
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO office_hours(id,assignment_id,kind,weekday,start_time,end_time,timezone,campus,room,meeting_mode,schedule_url,lecture_only,notes,sort_order) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, secret.New(), a.ID, h.Kind, weekday, start, end, h.Timezone, h.Campus, h.Room, h.MeetingMode, h.ScheduleURL, h.LectureOnly, h.Notes, i); err != nil {
			return "", err
		}
	}
	if err := access.Audit(ctx, tx, actor, a.ID, "team.assignment_saved"); err != nil {
		return "", err
	}
	return a.ID, tx.Commit()
}

func (s *Service) RemoveAssignment(ctx context.Context, actor, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := access.Require(ctx, tx, actor, access.ManageOrganization); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM team_assignments WHERE id=?`, id)
	if err != nil {
		return err
	}
	if err := changed(result); err != nil {
		return err
	}
	if err := access.Audit(ctx, tx, actor, id, "team.assignment_removed"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) Positions(ctx context.Context, actor string) ([]Position, error) {
	if err := access.Require(ctx, s.db, actor, access.ManageOrganization); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,organization_unit_id,title,contact_email,sort_order,active FROM team_positions ORDER BY sort_order,title,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	positions := []Position{}
	for rows.Next() {
		var p Position
		if err := rows.Scan(&p.ID, &p.UnitID, &p.Title, &p.ContactEmail, &p.SortOrder, &p.Active); err != nil {
			return nil, err
		}
		positions = append(positions, p)
	}
	return positions, rows.Err()
}

func readHours(ctx context.Context, tx *sql.Tx, id string) ([]OfficeHour, error) {
	rows, err := tx.QueryContext(ctx, `SELECT kind,COALESCE(weekday,0),COALESCE(start_time,''),COALESCE(end_time,''),timezone,campus,room,meeting_mode,schedule_url,lecture_only,notes FROM office_hours WHERE assignment_id=? ORDER BY sort_order,id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hours := []OfficeHour{}
	for rows.Next() {
		var h OfficeHour
		if err := rows.Scan(&h.Kind, &h.Weekday, &h.StartTime, &h.EndTime, &h.Timezone, &h.Campus, &h.Room, &h.MeetingMode, &h.ScheduleURL, &h.LectureOnly, &h.Notes); err != nil {
			return nil, err
		}
		hours = append(hours, h)
	}
	return hours, rows.Err()
}

func (s *Service) Assignments(ctx context.Context, actor string) ([]Assignment, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := access.Require(ctx, tx, actor, access.ManageOrganization); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,position_id,user_id,published,sort_order FROM team_assignments ORDER BY sort_order,id`)
	if err != nil {
		return nil, err
	}
	assignments := []Assignment{}
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.ID, &a.PositionID, &a.UserID, &a.Published, &a.SortOrder); err != nil {
			rows.Close()
			return nil, err
		}
		assignments = append(assignments, a)
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return nil, err
	}
	for i := range assignments {
		assignments[i].Hours, err = readHours(ctx, tx, assignments[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return assignments, tx.Commit()
}

// PublicMembers returns only published assignments of enabled users in active
// positions. No private staff account email is selected or used as a fallback.
func (s *Service) PublicMembers(ctx context.Context) ([]Member, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT a.id,p.id,u.name,p.title,o.id,o.name,p.contact_email FROM team_assignments a JOIN team_positions p ON p.id=a.position_id JOIN organization_units o ON o.id=p.organization_unit_id JOIN users u ON u.id=a.user_id WHERE a.published=1 AND p.active=1 AND u.disabled_at IS NULL ORDER BY o.name,p.sort_order,p.title,p.id,a.sort_order,u.name,a.id`)
	if err != nil {
		return nil, err
	}
	members := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.AssignmentID, &m.PositionID, &m.Name, &m.Title, &m.UnitID, &m.UnitName, &m.ContactEmail); err != nil {
			rows.Close()
			return nil, err
		}
		members = append(members, m)
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return nil, err
	}
	for i := range members {
		members[i].Hours, err = readHours(ctx, tx, members[i].AssignmentID)
		if err != nil {
			return nil, err
		}
	}
	return members, tx.Commit()
}
