package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/astawh/web/internal/config"
	"github.com/astawh/web/internal/database"
	"github.com/astawh/web/internal/email"
	"github.com/astawh/web/model/access"
	"github.com/astawh/web/model/organization"
	"github.com/astawh/web/model/users"
	"github.com/astawh/web/site"
)

type application struct {
	handler http.Handler
	db      *sql.DB
	cancel  context.CancelFunc
	done    chan struct{}
}

func newApplication(ctx context.Context, cfg config.Config) (*application, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	db, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	identity := users.New(db)
	handler, err := site.New(site.Services{
		Organization: organization.New(cfg.SiteName),
		Access:       access.New(db), Users: identity,
		BuildVersion:  Version,
		BuildRevision: Revision,
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	var sender users.InvitationSender
	if cfg.SMTP.Address != "" {
		transport, err := email.New(cfg.SMTP)
		if err != nil {
			db.Close()
			return nil, err
		}
		origin, _ := url.Parse(cfg.PublicURL)
		sender = invitationSender{transport: transport, origin: origin.Scheme + "://" + origin.Host}
	}
	workerCtx, cancel := context.WithCancel(ctx)
	app := &application{handler: handler, db: db, cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(app.done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			if err := identity.Cleanup(workerCtx); err != nil && workerCtx.Err() == nil {
				log.Print("Session cleanup failed")
			}
			if sender != nil {
				if err := identity.DeliverInvitations(workerCtx, sender); err != nil && workerCtx.Err() == nil {
					log.Print("Invitation email delivery failed; queued messages will retry")
				}
			}
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return app, nil
}

func (a *application) Close() error { a.cancel(); <-a.done; return a.db.Close() }

type invitationSender struct {
	transport *email.Sender
	origin    string
}

func (s invitationSender) SendInvitation(ctx context.Context, m users.InvitationMail) error {
	link := s.origin + "/einladung?token=" + url.QueryEscape(m.Token)
	body := "Hallo " + m.Name + ",\n\ndu wurdest zum AStA-Zugang eingeladen. Über diesen Link kannst du dein Passwort festlegen:\n\n" + link + "\n\nDer Link ist sieben Tage gültig und kann nur einmal verwendet werden.\nWenn du diese Einladung nicht erwartet hast, kannst du sie ignorieren."
	return s.transport.Send(ctx, m.Recipient, "Einladung zum AStA-Zugang", body)
}
