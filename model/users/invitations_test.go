package users_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/astawh/web/internal/secret"
	"github.com/astawh/web/internal/testdb"
	"github.com/astawh/web/model/access"
	"github.com/astawh/web/model/users"
)

func pendingToken(t *testing.T, db *sql.DB, id string) string {
	t.Helper()
	var token string
	if err := db.QueryRow(`SELECT token FROM mail_outbox WHERE invitation_id=?`, id).Scan(&token); err != nil {
		t.Fatal(err)
	}
	return token
}

func TestInvitationAcceptanceIsAtomicAndSingleUse(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t)
	s := users.New(db)
	a := access.New(db)
	testdb.User(t, db, "admin", "administrator")
	unit, err := a.CreateUnit(ctx, "admin", "Finanzen", "department")
	if err != nil {
		t.Fatal(err)
	}
	input := users.InvitationInput{Name: "Anna", Email: " ANNA@example.org ", Grants: access.Grants{Roles: []string{"application_handler"}, Memberships: []access.Membership{{UnitID: unit, Access: "manage"}}}}
	id, err := s.Invite(ctx, "admin", input)
	if err != nil {
		t.Fatal(err)
	}
	token := pendingToken(t, db, id)
	var hash string
	if err := db.QueryRow(`SELECT token_hash FROM invitations WHERE id=?`, id).Scan(&hash); err != nil || hash == token || hash != secret.Hash(token) {
		t.Fatalf("token not fingerprinted: %v", err)
	}
	if _, err := s.AcceptInvitation(ctx, token, "short"); !errors.Is(err, access.ErrInvalid) {
		t.Fatalf("weak password accepted: %v", err)
	}
	u, err := s.AcceptInvitation(ctx, token, "a sufficiently long password")
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "anna@example.org" {
		t.Fatal(u.Email)
	}
	g, err := a.UserGrants(ctx, "admin", u.ID)
	if err != nil || len(g.Roles) != 1 || len(g.Memberships) != 1 || g.Memberships[0].UnitID != unit {
		t.Fatalf("missing grants: %+v %v", g, err)
	}
	if _, err := s.AcceptInvitation(ctx, token, "a sufficiently long password"); !errors.Is(err, users.ErrInvitation) {
		t.Fatalf("replayed token: %v", err)
	}
	var queued int
	if err := db.QueryRow(`SELECT count(*) FROM mail_outbox`).Scan(&queued); err != nil || queued != 0 {
		t.Fatalf("outbox not removed: %d %v", queued, err)
	}
	if _, err := s.Invite(ctx, "admin", input); !errors.Is(err, access.ErrConflict) {
		t.Fatalf("existing user invited: %v", err)
	}
}

func TestInvitationRevocationExpiryAndResend(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t)
	s := users.New(db)
	testdb.User(t, db, "admin", "administrator")
	input := users.InvitationInput{Name: "Anna", Email: "anna@example.org"}
	id, err := s.Invite(ctx, "admin", input)
	if err != nil {
		t.Fatal(err)
	}
	old := pendingToken(t, db, id)
	if _, err := s.Invite(ctx, "admin", input); !errors.Is(err, access.ErrConflict) {
		t.Fatal("duplicate pending invitation accepted")
	}
	if err := s.ResendInvitation(ctx, "admin", id); err != nil {
		t.Fatal(err)
	}
	if token := pendingToken(t, db, id); token == old {
		t.Fatal("resend reused token")
	}
	if _, err := s.AcceptInvitation(ctx, old, "a sufficiently long password"); !errors.Is(err, users.ErrInvitation) {
		t.Fatal("rotated token accepted")
	}
	token := pendingToken(t, db, id)
	if err := s.RevokeInvitation(ctx, "admin", id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AcceptInvitation(ctx, token, "a sufficiently long password"); !errors.Is(err, users.ErrInvitation) {
		t.Fatal("revoked token accepted")
	}
	id, err = s.Invite(ctx, "admin", input)
	if err != nil {
		t.Fatal(err)
	}
	token = pendingToken(t, db, id)
	if _, err := db.Exec(`UPDATE invitations SET created_at=?,expires_at=? WHERE id=?`, time.Now().Add(-2*time.Hour).Unix(), time.Now().Add(-time.Hour).Unix(), id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AcceptInvitation(ctx, token, "a sufficiently long password"); !errors.Is(err, users.ErrInvitation) {
		t.Fatal("expired token accepted")
	}
	if _, err := s.Invite(ctx, "admin", input); err != nil {
		t.Fatalf("expired invitation blocked replacement: %v", err)
	}
}

func TestInvitationAuthorizationAndRollback(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t)
	s := users.New(db)
	testdb.User(t, db, "admin", "administrator")
	testdb.User(t, db, "editor", "content_editor")
	input := users.InvitationInput{Name: "Anna", Email: "anna@example.org", Grants: access.Grants{Roles: []string{"administrator"}}}
	if _, err := s.Invite(ctx, "editor", input); !errors.Is(err, access.ErrForbidden) {
		t.Fatal("editor invited administrator")
	}
	input.Grants.Roles = []string{"unknown"}
	if _, err := s.Invite(ctx, "admin", input); !errors.Is(err, access.ErrInvalid) {
		t.Fatal("unknown role accepted")
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM invitations`).Scan(&count); err != nil || count != 0 {
		t.Fatal("failed invite persisted")
	}
	input.Grants.Roles = []string{"content_editor"}
	id, err := s.Invite(ctx, "admin", input)
	if err != nil {
		t.Fatal(err)
	}
	token := pendingToken(t, db, id)
	if _, err := db.Exec(`DELETE FROM user_roles WHERE user_id='admin'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AcceptInvitation(ctx, token, "a sufficiently long password"); !errors.Is(err, users.ErrInvitation) {
		t.Fatalf("revoked inviter granted access: %v", err)
	}
}

type sender struct {
	fail bool
	sent []users.InvitationMail
}

func (s *sender) SendInvitation(_ context.Context, m users.InvitationMail) error {
	if s.fail {
		return errors.New("smtp unavailable")
	}
	s.sent = append(s.sent, m)
	return nil
}

func TestInvitationMailRetriesAndRechecksEligibility(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t)
	s := users.New(db)
	testdb.User(t, db, "admin", "administrator")
	id, err := s.Invite(ctx, "admin", users.InvitationInput{Name: "Anna", Email: "anna@example.org"})
	if err != nil {
		t.Fatal(err)
	}
	token := pendingToken(t, db, id)
	mail := &sender{fail: true}
	if err := s.DeliverInvitations(ctx, mail); err == nil {
		t.Fatal("failed SMTP delivery acknowledged")
	}
	var attempts int
	if err := db.QueryRow(`SELECT attempts FROM mail_outbox`).Scan(&attempts); err != nil || attempts != 1 {
		t.Fatal("message not retained")
	}
	mail.fail = false
	if err := s.DeliverInvitations(ctx, mail); err != nil {
		t.Fatal(err)
	}
	if len(mail.sent) != 0 {
		t.Fatal("retry backoff ignored")
	}
	if _, err := db.Exec(`UPDATE mail_outbox SET next_attempt_at=0`); err != nil {
		t.Fatal(err)
	}
	if err := s.DeliverInvitations(ctx, mail); err != nil {
		t.Fatal(err)
	}
	if len(mail.sent) != 1 || mail.sent[0].Token != token {
		t.Fatal("wrong invitation delivered")
	}
	if err := s.ResendInvitation(ctx, "admin", id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE users SET disabled_at=1 WHERE id='admin'`); err != nil {
		t.Fatal(err)
	}
	if err := s.DeliverInvitations(ctx, mail); err != nil {
		t.Fatal(err)
	}
	if len(mail.sent) != 1 {
		t.Fatal("disabled inviter's message delivered")
	}
}

func TestFailedOutboxWriteRollsBackInvitation(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t)
	s := users.New(db)
	testdb.User(t, db, "admin", "administrator")
	if _, err := db.Exec(`CREATE TRIGGER fail_mail BEFORE INSERT ON mail_outbox BEGIN SELECT RAISE(ABORT,'queue unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	_, err := s.Invite(ctx, "admin", users.InvitationInput{Name: "Anna", Email: "anna@example.org", Grants: access.Grants{Roles: []string{"application_handler"}}})
	if err == nil {
		t.Fatal("invitation succeeded without queued mail")
	}
	for _, table := range []string{"invitations", "invitation_roles", "invitation_memberships", "access_events"} {
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial state in %s: %d %v", table, count, err)
		}
	}
}

func TestConcurrentInvitationAcceptance(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t)
	s := users.New(db)
	testdb.User(t, db, "admin", "administrator")
	id, err := s.Invite(ctx, "admin", users.InvitationInput{Name: "Anna", Email: "anna@example.org"})
	if err != nil {
		t.Fatal(err)
	}
	token := pendingToken(t, db, id)
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() { _, err := s.AcceptInvitation(ctx, token, "a sufficiently long password"); results <- err })
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, users.ErrInvitation) {
			t.Fatalf("unexpected acceptance failure: %v", err)
		}
	}
	if success != 1 {
		t.Fatalf("successful acceptances: %d", success)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM users WHERE email='anna@example.org'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate users: %d %v", count, err)
	}
}

type senderFunc func(context.Context, users.InvitationMail) error

func (f senderFunc) SendInvitation(ctx context.Context, m users.InvitationMail) error {
	return f(ctx, m)
}

func TestMailWorkersDoNotShareALease(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t)
	s := users.New(db)
	testdb.User(t, db, "admin", "administrator")
	if _, err := s.Invite(ctx, "admin", users.InvitationInput{Name: "Anna", Email: "anna@example.org"}); err != nil {
		t.Fatal(err)
	}
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- s.DeliverInvitations(ctx, senderFunc(func(context.Context, users.InvitationMail) error { close(started); <-release; return nil }))
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("first worker did not claim mail")
	}
	second := &sender{}
	err := s.DeliverInvitations(ctx, second)
	close(release)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(second.sent) != 0 {
		t.Fatal("second worker sent leased invitation")
	}
}
