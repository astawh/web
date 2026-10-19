package site

import (
	"log"
	"net/http"

	"github.com/astawh/web/view"
)

func (h *Handler) routes() http.Handler {
	mux := http.NewServeMux()
	h.apiRoutes(mux)
	mux.HandleFunc("GET /{$}", h.home)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(view.Assets())))
	return mux
}

func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	org := h.services.Organization.Public()
	data, err := h.views.Render(view.Page{
		Title: "Startseite", SiteName: org.Name,
		BuildVersion: h.services.BuildVersion, BuildRevision: h.services.BuildRevision,
	})
	if err != nil {
		log.Printf("Render homepage: %v", err)
		http.Error(w, "Die Seite ist gerade nicht verfügbar.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}
