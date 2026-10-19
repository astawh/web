package site

import (
	"net/http"

	"github.com/astawh/web/view"
)

// Initial public directory transcribed from asta-wh.de/team on 2026-10-19.
// This editorial snapshot does not create staff accounts or grant permissions.
func (h *Handler) team(w http.ResponseWriter, r *http.Request) {
	groups := []view.TeamGroup{
		{ID: "vorstand", Title: "Vorstand", Roles: []view.TeamRole{
			{Title: "Vorsitz", Email: "vorsitz@asta-wh.de", Members: []view.TeamMember{
				{Name: "Anil Özgür Ölmez", Campus: "Gelsenkirchen", Room: "B2.1.29", OfficeHours: "Nach Vereinbarung"},
			}},
			{Title: "Finanzen", Email: "finanzen@asta-wh.de", Members: []view.TeamMember{
				{Name: "Yakup Kaya", Campus: "Bocholt", Room: "A2.1.03", OfficeHours: "Donnerstag, 13:00–14:00 Uhr"},
			}},
			{Title: "Standortkoordination Recklinghausen", Email: "StandortkoordinatorRE@asta-wh.de", Members: []view.TeamMember{
				{Name: "Lisa Scharfenberg", Campus: "Recklinghausen", Room: "B7.1.305", OfficeHours: "Mittwoch, 14:00–15:00 Uhr"},
			}},
			{Title: "Standortkoordination Gelsenkirchen", Email: "StandortkoordinatorGE@asta-wh.de", Members: []view.TeamMember{
				{Name: "Enes Sekili", Campus: "Gelsenkirchen", Room: "B2.1.29", OfficeHours: "Montag, 14:00–15:00 Uhr"},
			}},
		}},
		{ID: "referate", Title: "Referate", Roles: []view.TeamRole{
			{Title: "Mobilität und Nachhaltigkeit", Email: "mobilitaet@asta-wh.de", Members: []view.TeamMember{
				{Name: "Christian Kellmann", Campus: "Gelsenkirchen", Room: "B2.1.29", OfficeHours: "Mittwoch, 13:00–14:00 Uhr"},
			}},
			{Title: "Öffentlichkeitsarbeit", Email: "oeffentlichkeit@asta-wh.de", Members: []view.TeamMember{
				{Name: "Yannic Leismann", Campus: "Gelsenkirchen", Room: "B2.1.29", OfficeHours: "Mittwoch, 10:30–11:30 Uhr"},
			}},
			{Title: "EDV", Email: "edv@asta-wh.de", Members: []view.TeamMember{
				{Name: "Francesco La Camera", Campus: "Gelsenkirchen", Room: "B2.1.29", OfficeHours: "Freitag, 11:30–12:30 Uhr"},
				{Name: "Rivaldo Tandah Demlabing", Campus: "Gelsenkirchen", Room: "B2.1.29", OfficeHours: "Dienstag, 12:30–13:30 Uhr"},
			}},
			{Title: "Soziales und Integration", Email: "soziales@asta-wh.de", Members: []view.TeamMember{
				{Name: "Alejandro Abazi", Campus: "Gelsenkirchen", Room: "B2.1.29", OfficeHours: "Montag, 09:00–10:00 Uhr · via Teams"},
				{Name: "Frederike Winkelmann", Campus: "Recklinghausen", Room: "B7.1.305", OfficeHours: "Mittwoch, 12:00–13:00 Uhr"},
				{Name: "Enisa Abdurahmani", Campus: "Bocholt", Room: "A2.1.03", OfficeHours: "Mittwoch, 13:00–14:00 Uhr"},
			}},
			{Title: "Sport", Email: "sport@asta-wh.de", Members: []view.TeamMember{
				{Name: "Mohamed Sebai", Campus: "Bocholt", Room: "A2.1.03", OfficeHours: "Donnerstag, 12:00–13:00 Uhr"},
			}},
			{Title: "Kultur", Email: "kultur@asta-wh.de", Members: []view.TeamMember{
				{Name: "Christopher Pollak", Campus: "Gelsenkirchen", Room: "B2.1.29", OfficeHours: "Montag, 15:00–16:00 Uhr"},
				{Name: "Anna Feller", Campus: "Gelsenkirchen", Room: "B2.1.29", OfficeHours: "Nach Vereinbarung"},
				{Name: "Alina Ashkhharyan", Campus: "Recklinghausen", Room: "B7.1.305", OfficeHours: "Dienstag, 09:45–10:45 Uhr"},
				{Name: "Enya Könighaus", Campus: "Bocholt", Room: "A2.1.03", OfficeHours: "Dienstag, 12:30–13:30 Uhr"},
				{Name: "Stanislav Dyakov", Campus: "Recklinghausen", Room: "B7.1.305", OfficeHours: "Freitag, 13:00–14:00 Uhr"},
			}},
		}},
		{ID: "verwaltung", Title: "Verwaltung", Roles: []view.TeamRole{
			{Title: "Shopmanagement und Buchhaltung", Email: "verwaltung@asta-wh.de", Members: []view.TeamMember{
				{Name: "Christiane Rausch", Campus: "Gelsenkirchen", Room: "B2.1.28", Schedule: view.NavigationLink{Title: "Öffnungszeiten des Shops", URL: "https://www.asta-wh.de/shop/"}},
			}},
		}},
	}
	page := view.Page{
		Title: "Team und Sprechzeiten", Section: "team", Breadcrumbs: committeeBreadcrumbs(),
		Introduction: "Deine Ansprechpersonen im AStA: Zuständigkeiten, Kontakt und Sprechzeiten.",
		Team:         groups,
	}
	for _, group := range groups {
		page.Contents = append(page.Contents, view.NavigationLink{Title: group.Title, URL: "#" + group.ID})
	}
	h.render(w, page)
}
