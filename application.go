package main

import (
	"net/http"

	"github.com/astawh/web/internal/config"
	"github.com/astawh/web/model/organization"
	"github.com/astawh/web/site"
)

func newApplication(cfg config.Config) (http.Handler, error) {
	return site.New(site.Services{
		Organization:  organization.New(cfg.SiteName),
		BuildVersion:  Version,
		BuildRevision: Revision,
	})
}
