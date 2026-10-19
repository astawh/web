// Package view renders presentation-only data using embedded templates.
package view

type Page struct {
	Title, SiteName             string
	BuildVersion, BuildRevision string
	Section, Query              string
	Navigation                  []NavigationGroup
	Menu                        []NavigationGroup
	Results                     []NavigationLink
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
