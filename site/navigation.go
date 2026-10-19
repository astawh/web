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
			{Title: "Hochschulpolitik und Gremien", URL: "/hochschulpolitik"},
			{Title: "So funktioniert der AStA", URL: "/asta/so-funktioniert-der-asta"},
			{Title: "Team und Vertretung", URL: "https://www.asta-wh.de/team/"},
			{Title: "Aktuelles", URL: "https://www.asta-wh.de/"},
			{Title: "Standorte und Kontakt", URL: "https://www.asta-wh.de/kontakt/"},
			{Title: "Formulare und Downloads", URL: "/formulare"},
		}},
	}
}

func footerMenu() []view.NavigationGroup {
	return []view.NavigationGroup{
		{Title: "Angebote und Informationen", Links: []view.NavigationLink{
			{Title: "Semesterticket", URL: "https://www.asta-wh.de/semesterticket/"},
			{Title: "Beratung im Studium", URL: "https://www.asta-wh.de/beratung/"},
			{Title: "Rechtsberatung", URL: "https://www.asta-wh.de/rechtsberatung/"},
			{Title: "Finanzielle Unterstützung", URL: "https://www.asta-wh.de/asta-darlehen/"},
			{Title: "Sport und Fitness", URL: "https://www.asta-wh.de/fitnesskeller/"},
			{Title: "Shop", URL: "https://www.asta-wh.de/shop/"},
			{Title: "Skripteportal", URL: "https://skripte.asta-wh.de/"},
			{Title: "Schließfächer", URL: "https://www.asta-wh.de/schliessfaecher/"},
			{Title: "Vergünstigungen und Kooperationen", URL: "https://www.asta-wh.de/kooperationen/"},
			{Title: "Hochschulgruppen", URL: "https://www.asta-wh.de/hochschulgruppen/"},
			{Title: "Formulare und Downloads", URL: "/formulare"},
			{Title: "Alle Themen", URL: "/navigation"},
		}},
		{Title: "Über den AStA", Links: []view.NavigationLink{
			{Title: "Hochschulpolitik und Gremien", URL: "/hochschulpolitik"},
			{Title: "Team und Referate", URL: "https://www.asta-wh.de/team/"},
			{Title: "Aktuelles", URL: "https://www.asta-wh.de/"},
			// Planned destinations for the public information architecture.
			{Title: "Beschlüsse und Protokolle", URL: "/asta/protokolle"},
			{Title: "Finanzen und Transparenz", URL: "/asta/transparenz"},
			{Title: "Studierendenparlament", URL: "https://www.stupa-wh.de/"},
			{Title: "So funktioniert der AStA", URL: "/asta/so-funktioniert-der-asta"},
			{Title: "Mitmachen", URL: "/asta/mitmachen"},
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
			{Title: "Hochschulpolitik und Gremien", URL: "/hochschulpolitik", Description: "Studentische Selbstverwaltung, AStA und Studierendenparlament"},
			{Title: "So funktioniert der AStA", URL: "/asta/so-funktioniert-der-asta", Description: "Aufgaben, Vorstand und Referate, Studierendenparlament, Finanzierung und Mitmachen"},
			{Title: "Team und Sprechzeiten", URL: "https://www.asta-wh.de/team/", Description: "Deine Ansprechpersonen und ihre Sprechzeiten"},
			{Title: "Kontakt", URL: "https://www.asta-wh.de/kontakt/", Description: "So erreichst du uns"},
			{Title: "Gelsenkirchen", URL: "https://www.asta-wh.de/category/gelsenkirchen/", Description: "Aktuelles von deinem Standort"},
			{Title: "Recklinghausen", URL: "https://www.asta-wh.de/category/recklinghausen/", Description: "Aktuelles von deinem Standort"},
			{Title: "Bocholt", URL: "https://www.asta-wh.de/category/bocholt/", Description: "Aktuelles von deinem Standort"},
			{Title: "Studierendenparlament", URL: "https://www.stupa-wh.de/", Description: "Das StuPa der Westfälischen Hochschule"},
		}},
	}
}
