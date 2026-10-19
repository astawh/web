package database_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/astawh/web/internal/database"
	"github.com/astawh/web/internal/datetime"
)

func TestAllTimestampColumnsEnforceUTC(t *testing.T) {
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "sqlite.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	const initial = "2026-10-19T00:00:00Z"
	const expiry = "2026-10-20T00:00:00Z"
	for _, statement := range []string{
		`INSERT INTO users VALUES('u','User','user@example.org','hash','` + initial + `',NULL,'` + initial + `','` + initial + `')`,
		`INSERT INTO organization_units VALUES('unit','Finance','department','` + initial + `')`,
		`INSERT INTO invitations(id,name,email,token_hash,invited_by,created_at,expires_at) VALUES('invite','Invitee','invitee@example.org','hash','u','` + initial + `','` + expiry + `')`,
		`INSERT INTO sessions VALUES('session','u','` + initial + `','` + expiry + `')`,
		`INSERT INTO mail_outbox(id,invitation_id,recipient,token,created_at,next_attempt_at) VALUES('mail','invite','invitee@example.org','raw-token','` + initial + `','` + initial + `')`,
		`INSERT INTO access_events(actor_id,subject_id,action,created_at) VALUES('u','u','test','` + initial + `')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	// Discover every timestamp, including migration metadata and nullable fields,
	// so a newly added unconstrained timestamp fails this contract test.
	rows, err := db.Query(`SELECT m.name,p.name FROM sqlite_schema m JOIN pragma_table_info(m.name) p WHERE m.type='table' ORDER BY m.name,p.cid`)
	if err != nil {
		t.Fatal(err)
	}
	type column struct{ table, name string }
	var columns []column
	for rows.Next() {
		var c column
		if err := rows.Scan(&c.table, &c.name); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(c.name, "_at") || strings.HasSuffix(c.name, "_until") || c.name == "tstamp" {
			columns = append(columns, c)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if len(columns) != 16 {
		t.Fatalf("unexpected timestamp inventory: %d", len(columns))
	}
	invalid := []any{
		int64(1792368000), "", "2026-10-19", "2026-10-19 00:00:00", "2026-10-19T00:00:00",
		"2026-10-19T00:00:00+00:00", "2026-10-19T02:00:00+02:00", "2026-10-19T00:00:00.000Z",
		"2026-10-19T00:00:00z", "2026-02-30T00:00:00Z", "2026-10-19T24:00:00Z", "2026-10-19T00:00:60Z",
		"2026-10-19T00:00:00Z\n", "not-a-date",
	}
	for _, c := range columns {
		t.Run(c.table+"."+c.name, func(t *testing.T) {
			query := `UPDATE "` + c.table + `" SET "` + c.name + `"=?`
			for _, value := range invalid {
				if _, err := db.Exec(query, value); err == nil {
					t.Fatalf("accepted noncanonical timestamp: %#v", value)
				} else if !strings.Contains(err.Error(), "CHECK constraint failed") {
					t.Fatalf("unexpected rejection: %v", err)
				}
			}
			tx, err := db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if _, err := tx.Exec(query, "2026-10-19T00:01:00Z"); err != nil {
				t.Fatalf("valid UTC rejected: %v", err)
			}
		})
	}
	var stored string
	if err := db.QueryRow(`SELECT CAST(tstamp AS TEXT) FROM goose_db_version LIMIT 1`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	parsed, err := time.Parse(time.RFC3339, stored)
	if err != nil || datetime.Format(parsed) != stored {
		t.Fatalf("noncanonical migration timestamp: %q %v", stored, err)
	}
}
