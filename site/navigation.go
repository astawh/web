package site

import "github.com/astawh/web/view"

// Keep the compact, task-oriented menu from the initial layout prototype.
// The complete catalogue remains available through the topic index and search.
func headerMenu() []view.NavigationGroup {
	return []view.NavigationGroup{
		{Title: "Angebote und Informationen", Links: []view.NavigationLink{
			{Title: "Semesterticket", URL: "https://www.asta-wh.de/semesterticket/"},
			{Title: "Beratung", URL: "https://www.asta-wh.de/beratung/"},
			{Title: "Finanzielle Unterstützung", URL: "https://www.asta-wh.de/asta-darlehen/"},
			{Title: "Sport und Freizeit", URL: "https://www.asta-wh.de/fitnesskeller/"},
			{Title: "Skripteportal", URL: "https://skripte.asta-wh.de/"},
			{Title: "Shop", URL: "https://www.asta-wh.de/shop/"},
		}},
		{Title: "Dein AStA", Links: []view.NavigationLink{
			{Title: "Team und Vertretung", URL: "https://www.asta-wh.de/team/"},
			{Title: "Aktuelles", URL: "https://www.asta-wh.de/"},
			{Title: "Standorte und Kontakt", URL: "https://www.asta-wh.de/kontakt/"},
			{Title: "Formulare und Downloads", URL: "/formulare"},
		}},
	}
}

// Public destinations stay on the existing site until their new SSR pages land.
func navigation() []view.NavigationGroup {
	return []view.NavigationGroup{
		{Title: "Angebote und Informationen", Links: []view.NavigationLink{
			{Title: "Fitnesskeller", URL: "https://www.asta-wh.de/fitnesskeller/", Description: "Sport und Training an der Hochschule"},
			{Title: "Skripteportal", URL: "https://skripte.asta-wh.de", Description: "Skripte und Materialien für dein Studium"},
			{Title: "Shop", URL: "https://www.asta-wh.de/shop/", Description: "Angebot und Öffnungszeiten des Shops"},
			{Title: "AStA-Beratung", URL: "https://www.asta-wh.de/beratung/", Description: "Unterstützung bei Fragen und Problemen"},
			{Title: "AStA-Darlehen", URL: "https://www.asta-wh.de/asta-darlehen/", Description: "Finanzielle Unterstützung im Studium"},
			{Title: "Rechtsberatung", URL: "https://www.asta-wh.de/rechtsberatung/", Description: "Beratung bei rechtlichen Fragen"},
			{Title: "Semesterticket", URL: "https://www.asta-wh.de/semesterticket/", Description: "Informationen zur Mobilität im Studium"},
			{Title: "Kooperationen", URL: "https://www.asta-wh.de/kooperationen/", Description: "Partner und Vergünstigungen"},
			{Title: "Schließfächer", URL: "https://www.asta-wh.de/schliessfaecher/", Description: "Schließfächer an deinem Standort"},
			{Title: "Hochschulgruppen", URL: "https://www.asta-wh.de/hochschulgruppen/", Description: "Gemeinsam aktiv an der Hochschule"},
			{Title: "Formulare und Downloads", URL: "/formulare", Description: "Anmeldung und Kündigung für den Sportkeller, Veranstaltungsanträge und Checklisten"},
		}},
		{Title: "Dein AStA", Links: []view.NavigationLink{
			{Title: "Team und Sprechzeiten", URL: "https://www.asta-wh.de/team/", Description: "Deine Ansprechpersonen und ihre Sprechzeiten"},
			{Title: "Kontakt", URL: "https://www.asta-wh.de/kontakt/", Description: "So erreichst du uns"},
			{Title: "Gelsenkirchen", URL: "https://www.asta-wh.de/category/gelsenkirchen/", Description: "Aktuelles von deinem Standort"},
			{Title: "Recklinghausen", URL: "https://www.asta-wh.de/category/recklinghausen/", Description: "Aktuelles von deinem Standort"},
			{Title: "Bocholt", URL: "https://www.asta-wh.de/category/bocholt/", Description: "Aktuelles von deinem Standort"},
			{Title: "Studierendenparlament", URL: "https://www.stupa-wh.de/", Description: "Das StuPa der Westfälischen Hochschule"},
		}},
	}
}
