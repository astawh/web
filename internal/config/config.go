// Package config reads process configuration at the application boundary.
package config

import "os"

type Config struct{ SiteName string }

func New() Config {
	name := os.Getenv("SITE_NAME")
	if name == "" {
		name = "AStA WH"
	}
	return Config{SiteName: name}
}
