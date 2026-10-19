package site_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/astawh/web/internal/secret"
	"github.com/astawh/web/internal/testdb"
	"github.com/astawh/web/model/access"
	"github.com/astawh/web/model/organization"
	"github.com/astawh/web/model/users"
	"github.com/astawh/web/site"
)

func apiHandler(t *testing.T) (http.Handler, *sql.DB, *users.Service) {
	t.Helper()
	db := testdb.Open(t)
	identity := users.New(db)
	h, err := site.New(site.Services{Organization: organization.New("AStA WH"), Access: access.New(db), Users: identity, Origin: "https://asta.example"})
	if err != nil {
		t.Fatal(err)
	}
	return h, db, identity
}

func callAPI(h http.Handler, method, path, body string, cookie *http.Cookie, csrf, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "https://asta.example"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	if csrf != "" {
		r.Header.Set("X-CSRF-Token", csrf)
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestInvitationAPIEndToEnd(t *testing.T) {
	ctx := context.Background()
	h, db, identity := apiHandler(t)
	admin, err := identity.Bootstrap(ctx, "Administrator", "admin@example.org", "a sufficiently long password")
	if err != nil {
		t.Fatal(err)
	}
	w := callAPI(h, "POST", "/api/session", `{"email":"admin@example.org","password":"a sufficiently long password"}`, nil, "", "https://asta.example")
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	var session struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("missing cookie")
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("cookie policy")
	}
	if w := callAPI(h, "POST", "/api/admin/organization-units", `{"name":"Finanzen","kind":"department"}`, cookie, session.CSRF, ""); w.Code != 201 {
		t.Fatalf("create unit: %d %s", w.Code, w.Body.String())
	}
	var unit string
	if err := db.QueryRow(`SELECT id FROM organization_units`).Scan(&unit); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"name":"Anna","email":"anna@example.org","grants":{"roles":["application_handler"],"memberships":[{"unit_id":%q,"access":"manage"}]}}`, unit)
	w = callAPI(h, "POST", "/api/admin/invitations", body, cookie, session.CSRF, "")
	if w.Code != 201 {
		t.Fatalf("invite: %d %s", w.Code, w.Body.String())
	}
	var token string
	if err := db.QueryRow(`SELECT token FROM mail_outbox`).Scan(&token); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(w.Body.String(), token) {
		t.Fatal("invitation response disclosed token")
	}
	payload, _ := json.Marshal(map[string]string{"token": token, "password": "another sufficiently long password"})
	w = callAPI(h, "POST", "/api/invitations/accept", string(payload), nil, "", "")
	if w.Code != 201 {
		t.Fatalf("accept: %d %s", w.Code, w.Body.String())
	}
	var anna users.User
	if err := json.Unmarshal(w.Body.Bytes(), &anna); err != nil {
		t.Fatal(err)
	}
	g, err := access.New(db).UserGrants(ctx, admin.ID, anna.ID)
	if err != nil || len(g.Memberships) != 1 {
		t.Fatalf("membership: %+v %v", g, err)
	}
	w = callAPI(h, "DELETE", "/api/session", "", cookie, session.CSRF, "")
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	w = callAPI(h, "GET", "/api/admin/users", "", cookie, "", "")
	if w.Code != 401 {
		t.Fatal("logged-out session accepted")
	}
}

func TestAPIBoundaries(t *testing.T) {
	h, db, _ := apiHandler(t)
	testdb.User(t, db, "admin", "administrator")
	testdb.User(t, db, "editor", "content_editor")
	token := secret.New()
	if _, err := db.Exec(`INSERT INTO sessions VALUES(?,'admin',0,4102444800)`, secret.Hash(token)); err != nil {
		t.Fatal(err)
	}
	cookie := &http.Cookie{Name: "asta_session", Value: token}
	csrf := secret.Hash("csrf:" + token)
	for _, tc := range []struct {
		name, method, path, body, csrf, origin string
		cookie                                 *http.Cookie
		status                                 int
	}{
		{"anonymous", "GET", "/api/admin/users", "", "", "", nil, 401},
		{"no csrf", "POST", "/api/admin/invitations", `{}`, "", "", cookie, 403},
		{"wrong origin", "POST", "/api/admin/invitations", `{}`, csrf, "https://other.example", cookie, 403},
		{"unknown field", "POST", "/api/admin/invitations", `{"unexpected":true}`, csrf, "", cookie, 422},
		{"trailing document", "POST", "/api/admin/invitations", `{} {}`, csrf, "", cookie, 422},
		{"too large", "POST", "/api/admin/invitations", `{"name":"` + strings.Repeat("a", 70<<10) + `"}`, csrf, "", cookie, 422},
		{"no public signup", "POST", "/api/register", `{}`, "", "", nil, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := callAPI(h, tc.method, tc.path, tc.body, tc.cookie, tc.csrf, tc.origin)
			if w.Code != tc.status {
				t.Fatalf("got %d want %d: %s", w.Code, tc.status, w.Body.String())
			}
		})
	}
	r := httptest.NewRequest("POST", "/api/admin/invitations", bytes.NewBufferString(`{}`))
	r.AddCookie(cookie)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 415 {
		t.Fatal("simple cross-site content type accepted")
	}
	if _, err := db.Exec(`UPDATE sessions SET user_id='editor'`); err != nil {
		t.Fatal(err)
	}
	w = callAPI(h, "POST", "/api/admin/invitations", `{"name":"Other","email":"other@example.org"}`, cookie, csrf, "")
	if w.Code != 403 {
		t.Fatal("editor invited a user")
	}
	w = callAPI(h, "GET", "/api/session", "", cookie, "", "")
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("session response")
	}
	if _, err := db.Exec(`UPDATE users SET disabled_at=1 WHERE id='editor'`); err != nil {
		t.Fatal(err)
	}
	w = callAPI(h, "GET", "/api/session", "", cookie, "", "")
	if w.Code != 401 {
		t.Fatal("disabled account retained session")
	}
}

func TestAuthenticationRateLimit(t *testing.T) {
	h, _, _ := apiHandler(t)
	for range 20 {
		w := callAPI(h, "POST", "/api/session", `{"unexpected":true}`, nil, "", "")
		if w.Code != 422 {
			t.Fatal(w.Code)
		}
	}
	if w := callAPI(h, "POST", "/api/session", `{}`, nil, "", ""); w.Code != 429 {
		t.Fatalf("rate limit: %d", w.Code)
	}
}
