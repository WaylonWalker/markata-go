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
			Patterns:     []string{"content/**/*.md", "*.md"},
			UseGitignore: true,
		},
		Feeds: []models.FeedConfig{
			{
				Slug:        "posts",
				Title:       "All Posts",
				Description: "All posts from this site",
				Type:        models.FeedTypeBlog,
				Filter:      "published == true",
				Sort:        "date",
				Reverse:     true,
				Sidebar:     true,
				Formats: models.FeedFormats{
					HTML: true,
					RSS:  true,
				},
			},
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
