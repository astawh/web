// Package config reads process configuration at the application boundary.
package config

import (
	"errors"
	"net/url"
	"os"

	"github.com/astawh/web/internal/email"
)

type Config struct {
	SiteName  string
	Database  string
	PublicURL string
	SMTP      email.Config
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
	return Config{SiteName: name, Database: path, PublicURL: origin, SMTP: email.Config{
		Address: os.Getenv("SMTP_ADDRESS"), Username: os.Getenv("SMTP_USERNAME"), Password: os.Getenv("SMTP_PASSWORD"), From: os.Getenv("SMTP_FROM"), Mode: os.Getenv("SMTP_MODE"),
	}}
}

func (c Config) Validate() error {
	u, err := url.Parse(c.PublicURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return errors.New("PUBLIC_URL must be an absolute HTTP(S) origin")
	}
	if c.Database == "" {
		return errors.New("database path is required")
	}
	if c.SMTP.Address == "" && (c.SMTP.Username != "" || c.SMTP.Password != "" || c.SMTP.From != "" || c.SMTP.Mode != "") {
		return errors.New("SMTP_ADDRESS is required when SMTP settings are provided")
	}
	return nil
}
