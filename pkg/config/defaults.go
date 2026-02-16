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
			Patterns:     []string{"pages/**/*.md", "posts/**/*.md"},
			UseGitignore: true,
		},
		Feeds: []models.FeedConfig{
			{
				Slug:        "archive",
				Title:       "Archive",
				Description: "All posts",
				Filter:      "published == true",
				Sort:        "date",
				Reverse:     true,
			},
		},
		FeedDefaults: models.NewFeedDefaults(),
		WellKnown:    models.NewWellKnownConfig(),
		WebSub:       models.NewWebSubConfig(),
		Encryption:   models.NewEncryptionConfig(),
	}
}
