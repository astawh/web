package site

import (
	"net/http"

	"github.com/astawh/web/view"
)

func (h *Handler) forms(w http.ResponseWriter, r *http.Request) {
	registration := view.NavigationLink{
		Title:       "Sportkeller-Anmeldung (PDF)",
		URL:         "https://www.asta-wh.de/wp-content/uploads/2026/06/Anmeldung_Sportkeller.pdf",
		Description: "Für die Mitgliedschaft im Sportkeller anmelden.",
	}
	event := view.NavigationLink{
		Title:       "Veranstaltung anmelden (PDF)",
		URL:         "https://www.asta-wh.de/wp-content/uploads/2024/03/Dez-IV_Antrag-zur-Durchfuehrung-einer-Veranstaltung_rev03-2.pdf",
		Description: "Eine Veranstaltung an der Hochschule beantragen. Den Antrag grundsätzlich vier Wochen vor Veranstaltungsbeginn stellen; Fachschaftsveranstaltungen müssen vom AStA abgezeichnet werden.",
	}
	cancellation := view.NavigationLink{
		Title:       "Sportkeller-Mitgliedschaft kündigen (PDF)",
		URL:         "https://www.asta-wh.de/wp-content/uploads/2019/08/Sportkeller_kuendigung.pdf",
		Description: "Die Mitgliedschaft im Sportkeller schriftlich kündigen.",
	}
	h.render(w, view.Page{
		Title: "Formulare", Section: "forms",
		Featured: []view.NavigationLink{registration, event, cancellation},
		Forms: []view.NavigationGroup{
			{Title: "Sportkeller", Links: []view.NavigationLink{
				registration,
				{Title: "Nutzungsbedingungen Sportkeller (PDF)", URL: "https://www.asta-wh.de/wp-content/uploads/2018/08/Vertrag-Fitnessstudio-2.-Seite-Nutzungsbedingungen.pdf", Description: "Die Bedingungen für die Nutzung des Sportkellers nachlesen."},
				cancellation,
			}},
			{Title: "Veranstaltungen", Links: []view.NavigationLink{
				event,
				{Title: "Checkliste für Veranstaltungen (PDF)", URL: "https://www.asta-wh.de/wp-content/uploads/2023/02/Checkliste-fuer-Veranstaltungen.pdf", Description: "Die wichtigsten Schritte für die Planung und Durchführung einer Veranstaltung."},
			}},
		},
	})
}
