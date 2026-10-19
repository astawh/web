package site

import (
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/astawh/web/view"
)

func (h *Handler) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.landing)
	mux.HandleFunc("GET /navigation", h.navigation)
	mux.HandleFunc("GET /suche", h.search)
	mux.HandleFunc("GET /formulare", h.forms)
	mux.HandleFunc("GET /asta/so-funktioniert-der-asta", h.aboutAsta)
	mux.HandleFunc("GET /asta/team", h.team)
	mux.HandleFunc("GET /hochschulpolitik", h.politics)
	mux.HandleFunc("GET /hochschulpolitik/gremien", h.committees)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(view.Assets())))
	return mux
}

func (h *Handler) landing(w http.ResponseWriter, r *http.Request) {
	h.render(w, view.Page{Title: "Startseite"})
}

func (h *Handler) navigation(w http.ResponseWriter, r *http.Request) {
	h.render(w, view.Page{Title: "Alle Themen", Section: "navigation"})
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if !utf8.ValidString(query) || utf8.RuneCountInString(query) > 200 {
		http.Error(w, "Der Suchbegriff darf höchstens 200 Zeichen enthalten.", http.StatusBadRequest)
		return
	}
	page := view.Page{Title: "Suche", Section: "search", Query: query}
	if query != "" {
		words := strings.Fields(strings.ToLower(query))
		for _, group := range navigation() {
			for _, link := range group.Links {
				text := strings.ToLower(link.Title + " " + link.Description + " " + group.Title)
				matches := true
				for _, word := range words {
					if !strings.Contains(text, word) {
						matches = false
						break
					}
				}
				if matches {
					page.Results = append(page.Results, link)
				}
			}
		}
	}
	h.render(w, page)
}

func (h *Handler) render(w http.ResponseWriter, page view.Page) {
	org := h.services.Organization.Public()
	page.SiteName = org.Name
	page.BuildVersion = h.services.BuildVersion
	page.BuildRevision = h.services.BuildRevision
	page.Navigation = navigation()
	page.Menu = headerMenu()
	page.Footer = footerMenu()
	data, err := h.views.Render(page)
	if err != nil {
		log.Printf("Render page: %v", err)
		http.Error(w, "Die Seite ist gerade nicht verfügbar.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}
