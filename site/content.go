package site

import (
	"net/http"

	"github.com/astawh/web/view"
)

func (h *Handler) aboutAsta(w http.ResponseWriter, r *http.Request) {
	h.render(w, view.Page{
		Title: "So funktioniert der AStA", Section: "asta",
		Breadcrumbs:  committeeBreadcrumbs(),
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

func committeeBreadcrumbs() []view.NavigationLink {
	return []view.NavigationLink{
		{Title: "Startseite", URL: "/"},
		{Title: "Hochschulpolitik", URL: "/hochschulpolitik"},
		{Title: "Gremien", URL: "/hochschulpolitik/gremien"},
	}
}

func (h *Handler) politics(w http.ResponseWriter, r *http.Request) {
	h.render(w, view.Page{
		Title: "Hochschulpolitik", Section: "browse",
		Introduction: "Studentische Selbstverwaltung an der Westfälischen Hochschule: Wer dich vertritt und wie du dich einbringen kannst.",
		Breadcrumbs:  []view.NavigationLink{{Title: "Startseite", URL: "/"}},
		Topics: []view.NavigationLink{
			{Title: "Gremien", URL: "/hochschulpolitik/gremien", Description: "AStA und Studierendenparlament: Aufgaben und Zuständigkeiten der studentischen Vertretung."},
			{Title: "Mitmachen im AStA", URL: "https://www.stupa-wh.de/ausschreibungen", Description: "Ausschreibungen des StuPa für Positionen im AStA und Informationen zur Bewerbung."},
		},
	})
}

func (h *Handler) committees(w http.ResponseWriter, r *http.Request) {
	h.render(w, view.Page{
		Title: "Gremien", Section: "browse",
		Introduction: "AStA und Studierendenparlament übernehmen unterschiedliche Aufgaben in der studentischen Selbstverwaltung.",
		Breadcrumbs:  committeeBreadcrumbs()[:2],
		Topics: []view.NavigationLink{
			{Title: "So funktioniert der AStA", URL: "/asta/so-funktioniert-der-asta", Description: "Aufgaben, Vorstand und Referate, Angebote, Finanzierung und Mitmachen."},
			{Title: "Studierendenparlament (StuPa)", URL: "https://www.stupa-wh.de/das-stupa", Description: "Das höchste beschlussfassende Organ der Studierendenschaft: Wahl und Kontrolle des AStA, Haushalt und grundlegende Entscheidungen."},
		},
	})
}
