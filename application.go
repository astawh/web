package main

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/astawh/web/internal/config"
	"github.com/astawh/web/internal/database"
	"github.com/astawh/web/model/access"
	"github.com/astawh/web/model/organization"
	"github.com/astawh/web/model/users"
	"github.com/astawh/web/site"
)

type application struct {
	handler http.Handler
	db      *sql.DB
}

func newApplication(ctx context.Context, cfg config.Config) (*application, error) {
	db, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	handler, err := site.New(site.Services{
		Organization: organization.New(cfg.SiteName),
		Access:       access.New(db), Users: users.New(db),
		BuildVersion:  Version,
		BuildRevision: Revision,
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &application{handler: handler, db: db}, nil
}
