package site_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/astawh/web/internal/testdb"
	"github.com/astawh/web/model/access"
	"github.com/astawh/web/model/organization"
	"github.com/astawh/web/model/users"
	"github.com/astawh/web/site"
)

func TestPublicRoutes(t *testing.T) {
	db := testdb.Open(t)
	handler, err := site.New(site.Services{
		Organization: organization.New("AStA <WH>"),
		Access:       access.New(db), Users: users.New(db),
		BuildVersion: "v0.1.0", BuildRevision: "1234567890",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path      string
		status            int
		contentType, body string
	}{
		{"GET", "/", 200, "text/html", "AStA &lt;WH&gt;"},
		{"GET", "/healthz", 200, "text/plain", "ok\n"},
		{"GET", "/assets/site.css", 200, "text/css", ".asta-page"},
		{"GET", "/assets/govuk.css", 200, "text/css", ".govuk-footer"},
		{"GET", "/assets/site.js", 200, "text/javascript", "SuperNavigationMegaMenu"},
		{"GET", "/assets/logo.svg", 200, "image/svg+xml", "<svg"},
		{"GET", "/navigation", 200, "text/html", "Alle Themen"},
		{"GET", "/formulare", 200, "text/html", "Alle Formulare"},
		{"GET", "/asta/so-funktioniert-der-asta", 200, "text/html", "Auf dieser Seite"},
		{"GET", "/suche?q=Vorstand", 200, "text/html", "href=\"/asta/so-funktioniert-der-asta\""},
		{"GET", "/suche?q=Formulare", 200, "text/html", "href=\"/formulare\""},
		{"GET", "/suche?q=Semesterticket", 200, "text/html", "1 Ergebnis"},
		{"GET", "/suche?q=unbekannteswort", 200, "text/html", "Keine passenden Themen"},
		{"GET", "/suche?q=%3Cscript%3E", 200, "text/html", "&lt;script&gt;"},
		{"GET", "/suche?q=" + strings.Repeat("x", 201), 400, "text/plain", "höchstens 200"},
		{"GET", "/missing", 404, "text/plain", "404"},
		{"POST", "/", 405, "text/plain", "Method Not Allowed"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("status: %d", w.Code)
			}
			if !strings.HasPrefix(w.Header().Get("Content-Type"), tc.contentType) {
				t.Fatalf("content type: %s", w.Header().Get("Content-Type"))
			}
			if !strings.Contains(w.Body.String(), tc.body) {
				t.Fatalf("body: %s", w.Body.String())
			}
			if w.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("missing response policy")
			}
		})
	}
}
