// Package view renders presentation-only data using embedded templates.
package view

type Page struct {
	Title, SiteName             string
	BuildVersion, BuildRevision string
}

func (p Page) ShortRevision() string {
	if len(p.BuildRevision) > 7 {
		return p.BuildRevision[:7]
	}
	return p.BuildRevision
}
