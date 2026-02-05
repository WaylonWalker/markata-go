// Package config provides configuration loading and management for markata-go.
package config

import (
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() *models.Config {
	return &models.Config{
		OutputDir:    "output",
		TemplatesDir: "templates",
		AssetsDir:    "static",
		Hooks:        []string{"default"},
		GlobConfig: models.GlobConfig{
			Patterns:     []string{"**/*.md"},
			UseGitignore: true,
		},
		FeedDefaults: models.FeedDefaults{
			ItemsPerPage:    10,
			OrphanThreshold: 3,
			Formats: models.FeedFormats{
				HTML: true,
				RSS:  true,
			},
		},
		WellKnown: models.NewWellKnownConfig(),
		WebSub:    models.NewWebSubConfig(),
	}
}
