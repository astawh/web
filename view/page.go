// Package view renders presentation-only data using embedded templates.
package view

type Page struct {
	Title, SiteName             string
	BuildVersion, BuildRevision string
	Section, Query              string
	Navigation                  []NavigationGroup
	Menu                        []NavigationGroup
	Footer                      []NavigationGroup
	Results                     []NavigationLink
	Forms                       []NavigationGroup
	Featured                    []NavigationLink
	Introduction                string
	Contents                    []NavigationLink
	Breadcrumbs                 []NavigationLink
	Topics                      []NavigationLink
	Team                        []TeamGroup
}

type TeamGroup struct {
	ID, Title string
	Roles     []TeamRole
}

type TeamRole struct {
	Title, Email string
	Members      []TeamMember
}

type TeamMember struct {
	Name, Campus, Room, OfficeHours string
	Schedule                        NavigationLink
}

type NavigationLink struct{ Title, URL, Description string }
type NavigationGroup struct {
	Title string
	Links []NavigationLink
}

func (p Page) ShortRevision() string {
	if len(p.BuildRevision) > 7 {
		return p.BuildRevision[:7]
	}
	return p.BuildRevision
}
