package view

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
)

//go:embed templates/*.html assets/*
var files embed.FS

type Renderer struct{ templates *template.Template }

func NewRenderer() (*Renderer, error) {
	t, err := template.ParseFS(files, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Renderer{templates: t}, nil
}

func (r *Renderer) Render(page Page) ([]byte, error) {
	var b bytes.Buffer
	if err := r.templates.ExecuteTemplate(&b, "layout", page); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func Assets() fs.FS {
	assets, err := fs.Sub(files, "assets")
	if err != nil {
		panic(err)
	}
	return assets
}
