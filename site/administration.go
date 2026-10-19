package site

import (
	"net/http"

	"github.com/astawh/web/model/access"
	"github.com/astawh/web/model/users"
)

func (h *Handler) listInvitations(w http.ResponseWriter, r *http.Request, u users.User) {
	items, err := h.services.Users.Invitations(r.Context(), u.ID)
	if err != nil {
		h.apiError(w, err)
		return
	}
	h.writeJSON(w, 200, items)
}
func (h *Handler) invite(w http.ResponseWriter, r *http.Request, u users.User) {
	var input users.InvitationInput
	if err := decodeJSON(w, r, &input); err != nil {
		h.apiError(w, err)
		return
	}
	id, err := h.services.Users.Invite(r.Context(), u.ID, input)
	if err != nil {
		h.apiError(w, err)
		return
	}
	h.writeJSON(w, 201, map[string]string{"id": id})
}
func (h *Handler) resendInvitation(w http.ResponseWriter, r *http.Request, u users.User) {
	if err := h.services.Users.ResendInvitation(r.Context(), u.ID, r.PathValue("id")); err != nil {
		h.apiError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) revokeInvitation(w http.ResponseWriter, r *http.Request, u users.User) {
	if err := h.services.Users.RevokeInvitation(r.Context(), u.ID, r.PathValue("id")); err != nil {
		h.apiError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request, u users.User) {
	items, err := h.services.Access.Users(r.Context(), u.ID)
	if err != nil {
		h.apiError(w, err)
		return
	}
	h.writeJSON(w, 200, items)
}
func (h *Handler) userAccess(w http.ResponseWriter, r *http.Request, u users.User) {
	grants, err := h.services.Access.UserGrants(r.Context(), u.ID, r.PathValue("id"))
	if err != nil {
		h.apiError(w, err)
		return
	}
	h.writeJSON(w, 200, grants)
}
func (h *Handler) setUserAccess(w http.ResponseWriter, r *http.Request, u users.User) {
	var input struct {
		Grants   access.Grants `json:"grants"`
		Disabled bool          `json:"disabled"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.apiError(w, err)
		return
	}
	if err := h.services.Access.SetUserAccess(r.Context(), u.ID, r.PathValue("id"), input.Grants, input.Disabled); err != nil {
		h.apiError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request, u users.User) {
	items, err := h.services.Access.Roles(r.Context(), u.ID)
	if err != nil {
		h.apiError(w, err)
		return
	}
	h.writeJSON(w, 200, items)
}
func (h *Handler) listUnits(w http.ResponseWriter, r *http.Request, u users.User) {
	items, err := h.services.Access.Units(r.Context(), u.ID)
	if err != nil {
		h.apiError(w, err)
		return
	}
	h.writeJSON(w, 200, items)
}
func (h *Handler) createUnit(w http.ResponseWriter, r *http.Request, u users.User) {
	var input struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.apiError(w, err)
		return
	}
	id, err := h.services.Access.CreateUnit(r.Context(), u.ID, input.Name, input.Kind)
	if err != nil {
		h.apiError(w, err)
		return
	}
	h.writeJSON(w, 201, map[string]string{"id": id})
}
func (h *Handler) listForms(w http.ResponseWriter, r *http.Request, u users.User) {
	items, err := h.services.Access.Forms(r.Context(), u.ID)
	if err != nil {
		h.apiError(w, err)
		return
	}
	h.writeJSON(w, 200, items)
}
func (h *Handler) createForm(w http.ResponseWriter, r *http.Request, u users.User) {
	var input struct {
		Key   string   `json:"key"`
		Title string   `json:"title"`
		Units []string `json:"units"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.apiError(w, err)
		return
	}
	id, err := h.services.Access.CreateForm(r.Context(), u.ID, input.Key, input.Title, input.Units)
	if err != nil {
		h.apiError(w, err)
		return
	}
	h.writeJSON(w, 201, map[string]string{"id": id})
}
func (h *Handler) setFormResponsibilities(w http.ResponseWriter, r *http.Request, u users.User) {
	var input struct {
		Units  []string `json:"units"`
		Active bool     `json:"active"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		h.apiError(w, err)
		return
	}
	if err := h.services.Access.SetFormResponsibilities(r.Context(), u.ID, r.PathValue("id"), input.Units, input.Active); err != nil {
		h.apiError(w, err)
		return
	}
	w.WriteHeader(204)
}
