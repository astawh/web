package site

import (
	"net/http"

	"github.com/astawh/web/view"
)

func (h *Handler) aboutAsta(w http.ResponseWriter, r *http.Request) {
	h.render(w, view.Page{
		Title: "So funktioniert der AStA", Section: "asta",
		Introduction: "Der Allgemeine Studierendenausschuss (AStA) vertritt die Interessen der Studierenden an der Westfälischen Hochschule. Hier erfährst du, wie der AStA arbeitet, welche Rolle das Studierendenparlament spielt und wie du mitmachen kannst.",
		Contents: []view.NavigationLink{
			{Title: "Was ist der AStA?", URL: "#was-ist-der-asta"},
			{Title: "Vorstand und Referate", URL: "#vorstand-und-referate"},
			{Title: "AStA und Studierendenparlament", URL: "#asta-und-studierendenparlament"},
			{Title: "Angebote für Studierende", URL: "#angebote"},
			{Title: "Finanzierung und Transparenz", URL: "#finanzierung"},
			{Title: "Mitmachen und Kontakt", URL: "#mitmachen"},
		},
	})
}
