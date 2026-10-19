// Package config reads process configuration at the application boundary.
package config

import "os"

type Config struct {
	SiteName  string
	Database  string
	PublicURL string
}

func New() Config {
	name := os.Getenv("SITE_NAME")
	if name == "" {
		name = "AStA WH"
	}
	path := os.Getenv("DATABASE_PATH")
	if path == "" {
		path = "data/sqlite.db"
	}
	origin := os.Getenv("PUBLIC_URL")
	if origin == "" {
		origin = "http://127.0.0.1:8090"
	}
	return Config{SiteName: name, Database: path, PublicURL: origin}
}
