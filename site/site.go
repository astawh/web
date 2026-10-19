// Package site coordinates HTTP requests, domain services and presentation.
package site

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/astawh/web/model/access"
	"github.com/astawh/web/model/organization"
	"github.com/astawh/web/model/users"
	"github.com/astawh/web/view"
)

type Services struct {
	Organization                *organization.Service
	Access                      *access.Service
	Users                       *users.Service
	Origin                      string
	BuildVersion, BuildRevision string
}

type Handler struct {
	services      Services
	views         *view.Renderer
	router        http.Handler
	secureCookies bool
	limiter       authLimiter
}

func New(services Services) (*Handler, error) {
	if services.Organization == nil || services.Access == nil || services.Users == nil {
		return nil, errors.New("application services are required")
	}
	origin, err := url.Parse(services.Origin)
	if err != nil || origin.Host == "" || (origin.Scheme != "http" && origin.Scheme != "https") || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || (origin.Path != "" && origin.Path != "/") {
		return nil, errors.New("origin must be an absolute HTTP(S) origin")
	}
	services.Origin = origin.Scheme + "://" + origin.Host
	views, err := view.NewRenderer()
	if err != nil {
		return nil, err
	}
	h := &Handler{services: services, views: views, secureCookies: origin.Scheme == "https"}
	h.router = h.routes()
	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'self'")
	h.router.ServeHTTP(w, r)
}
