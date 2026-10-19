package site

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/astawh/web/internal/secret"
	"github.com/astawh/web/model/access"
	"github.com/astawh/web/model/users"
)

type rateEntry struct {
	start time.Time
	count int
}
type authLimiter struct {
	mu      sync.Mutex
	entries map[string]rateEntry
}

func (l *authLimiter) allow(remote string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.entries == nil {
		l.entries = map[string]rateEntry{}
	}
	ip, _, err := net.SplitHostPort(remote)
	if err != nil {
		ip = remote
	}
	now := time.Now()
	for key, e := range l.entries {
		if now.Sub(e.start) >= 15*time.Minute {
			delete(l.entries, key)
		}
	}
	e, exists := l.entries[ip]
	if !exists {
		if len(l.entries) >= 4096 {
			return false
		}
		e.start = now
	}
	if e.count >= 20 {
		return false
	}
	e.count++
	l.entries[ip] = e
	return true
}

type apiFunc func(http.ResponseWriter, *http.Request, users.User)

func (h *Handler) api(authenticated bool, fn apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		unsafe := r.Method != http.MethodGet && r.Method != http.MethodHead
		if unsafe {
			if origin := r.Header.Get("Origin"); (origin != "" && origin != h.services.Origin) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				h.apiError(w, access.ErrForbidden)
				return
			}
			if r.Method != http.MethodDelete {
				media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if err != nil || media != "application/json" {
					h.writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "application/json required"})
					return
				}
			}
		}
		var u users.User
		if authenticated {
			cookie, err := r.Cookie("asta_session")
			if err != nil {
				h.apiError(w, users.ErrCredentials)
				return
			}
			u, err = h.services.Users.Authenticate(r.Context(), cookie.Value)
			if err != nil {
				h.apiError(w, err)
				return
			}
			if unsafe {
				want := secret.Hash("csrf:" + cookie.Value)
				if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(want)) != 1 {
					h.apiError(w, access.ErrForbidden)
					return
				}
			}
		} else if unsafe && !h.limiter.allow(r.RemoteAddr) {
			w.Header().Set("Retry-After", "900")
			h.writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many attempts"})
			return
		}
		fn(w, r, u)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return access.ErrInvalid
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return access.ErrInvalid
	}
	return nil
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Print("Write API response failed")
	}
}

func (h *Handler) apiError(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "request failed"
	switch {
	case errors.Is(err, users.ErrCredentials):
		status, message = http.StatusUnauthorized, "authentication required or credentials invalid"
	case errors.Is(err, users.ErrInvitation):
		status, message = http.StatusBadRequest, "invitation invalid or unavailable"
	case errors.Is(err, access.ErrForbidden):
		status, message = http.StatusForbidden, "access denied"
	case errors.Is(err, access.ErrInvalid):
		status, message = http.StatusUnprocessableEntity, "invalid input"
	case errors.Is(err, access.ErrConflict):
		status, message = http.StatusConflict, "operation conflicts with current state"
	default:
		log.Printf("API request failed: %v", err)
	}
	h.writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) apiRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/session", h.api(false, h.login))
	mux.HandleFunc("GET /api/session", h.api(true, h.session))
	mux.HandleFunc("DELETE /api/session", h.api(true, h.logout))
	mux.HandleFunc("POST /api/invitations/accept", h.api(false, h.acceptInvitation))
	mux.HandleFunc("GET /api/admin/invitations", h.api(true, h.listInvitations))
	mux.HandleFunc("POST /api/admin/invitations", h.api(true, h.invite))
	mux.HandleFunc("POST /api/admin/invitations/{id}/resend", h.api(true, h.resendInvitation))
	mux.HandleFunc("POST /api/admin/invitations/{id}/revoke", h.api(true, h.revokeInvitation))
	mux.HandleFunc("GET /api/admin/users", h.api(true, h.listUsers))
	mux.HandleFunc("GET /api/admin/users/{id}/access", h.api(true, h.userAccess))
	mux.HandleFunc("PUT /api/admin/users/{id}/access", h.api(true, h.setUserAccess))
	mux.HandleFunc("GET /api/admin/roles", h.api(true, h.listRoles))
	mux.HandleFunc("GET /api/admin/organization-units", h.api(true, h.listUnits))
	mux.HandleFunc("POST /api/admin/organization-units", h.api(true, h.createUnit))
	mux.HandleFunc("GET /api/admin/forms", h.api(true, h.listForms))
	mux.HandleFunc("POST /api/admin/forms", h.api(true, h.createForm))
	mux.HandleFunc("PUT /api/admin/forms/{id}/responsibilities", h.api(true, h.setFormResponsibilities))
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request, _ users.User) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.apiError(w, err)
		return
	}
	u, token, err := h.services.Users.Login(r.Context(), input.Email, input.Password)
	if err != nil {
		h.apiError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "asta_session", Value: token, Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteStrictMode, MaxAge: int(users.SessionLifetime / time.Second)})
	h.writeJSON(w, http.StatusOK, map[string]any{"user": u, "csrf_token": secret.Hash("csrf:" + token)})
}

func (h *Handler) session(w http.ResponseWriter, r *http.Request, u users.User) {
	cookie, _ := r.Cookie("asta_session")
	g, err := h.services.Users.Grants(r.Context(), u.ID)
	if err != nil {
		h.apiError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"user": u, "grants": g, "csrf_token": secret.Hash("csrf:" + cookie.Value)})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request, _ users.User) {
	cookie, _ := r.Cookie("asta_session")
	if err := h.services.Users.Logout(r.Context(), cookie.Value); err != nil {
		h.apiError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "asta_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteStrictMode})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) acceptInvitation(w http.ResponseWriter, r *http.Request, _ users.User) {
	var input struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.apiError(w, err)
		return
	}
	u, err := h.services.Users.AcceptInvitation(r.Context(), input.Token, input.Password)
	if err != nil {
		h.apiError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, u)
}
