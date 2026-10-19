// Package site coordinates HTTP requests, domain services and presentation.
package site

import (
	"errors"
	"net/http"

	"github.com/astawh/web/model/organization"
	"github.com/astawh/web/view"
)

type Services struct {
	Organization                *organization.Service
	BuildVersion, BuildRevision string
}

type Handler struct {
	services Services
	views    *view.Renderer
	router   http.Handler
}

func New(services Services) (*Handler, error) {
	if services.Organization == nil {
		return nil, errors.New("organization service is required")
	}
	views, err := view.NewRenderer()
	if err != nil {
		return nil, err
	}
	h := &Handler{services: services, views: views}
	h.router = h.routes()
	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'self'")
	h.router.ServeHTTP(w, r)
}
