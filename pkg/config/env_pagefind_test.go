package config

import (
	"os"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestApplyEnvOverrides_PagefindSettings(t *testing.T) {
	tests := []struct {
		name         string
		envVars      map[string]string
		expectedFunc func(*models.Config) bool
	}{
		{
			name: "search_pagefind_auto_install true",
			envVars: map[string]string{
				"MARKATA_GO_SEARCH_PAGEFIND_AUTO_INSTALL": "true",
			},
			expectedFunc: func(c *models.Config) bool {
				if c.Extra == nil {
					return false
				}
				searchConfig, ok := c.Extra["search"].(models.SearchConfig)
				if !ok {
					return false
				}
				return searchConfig.Pagefind.AutoInstall != nil && *searchConfig.Pagefind.AutoInstall == true
			},
		},
		{
			name: "search_pagefind_auto_install false",
			envVars: map[string]string{
				"MARKATA_GO_SEARCH_PAGEFIND_AUTO_INSTALL": "false",
			},
			expectedFunc: func(c *models.Config) bool {
				if c.Extra == nil {
					return false
				}
				searchConfig, ok := c.Extra["search"].(models.SearchConfig)
				if !ok {
					return false
				}
				return searchConfig.Pagefind.AutoInstall != nil && *searchConfig.Pagefind.AutoInstall == false
			},
		},
		{
			name: "search_pagefind_cache_dir",
			envVars: map[string]string{
				"MARKATA_GO_SEARCH_PAGEFIND_CACHE_DIR": "/tmp/pagefind-cache",
			},
			expectedFunc: func(c *models.Config) bool {
				if c.Extra == nil {
					return false
				}
				searchConfig, ok := c.Extra["search"].(models.SearchConfig)
				if !ok {
					return false
				}
				return searchConfig.Pagefind.CacheDir == "/tmp/pagefind-cache"
			},
		},
		{
			name: "search_pagefind_version",
			envVars: map[string]string{
				"MARKATA_GO_SEARCH_PAGEFIND_VERSION": "v1.4.0",
			},
			expectedFunc: func(c *models.Config) bool {
				if c.Extra == nil {
					return false
				}
				searchConfig, ok := c.Extra["search"].(models.SearchConfig)
				if !ok {
					return false
				}
				return searchConfig.Pagefind.Version == "v1.4.0"
			},
		},
		{
			name: "search_pagefind_bundle_dir",
			envVars: map[string]string{
				"MARKATA_GO_SEARCH_PAGEFIND_BUNDLE_DIR": "_search",
			},
			expectedFunc: func(c *models.Config) bool {
				if c.Extra == nil {
					return false
				}
				searchConfig, ok := c.Extra["search"].(models.SearchConfig)
				if !ok {
					return false
				}
				return searchConfig.Pagefind.BundleDir == "_search"
			},
		},
		{
			name: "search_pagefind_verbose true",
			envVars: map[string]string{
				"MARKATA_GO_SEARCH_PAGEFIND_VERBOSE": "true",
			},
			expectedFunc: func(c *models.Config) bool {
				if c.Extra == nil {
					return false
				}
				searchConfig, ok := c.Extra["search"].(models.SearchConfig)
				if !ok {
					return false
				}
				return searchConfig.Pagefind.Verbose != nil && *searchConfig.Pagefind.Verbose == true
			},
		},
		{
			name: "multiple pagefind settings",
			envVars: map[string]string{
				"MARKATA_GO_SEARCH_PAGEFIND_AUTO_INSTALL": "true",
				"MARKATA_GO_SEARCH_PAGEFIND_CACHE_DIR":    "/tmp/cache",
				"MARKATA_GO_SEARCH_PAGEFIND_VERSION":      "v1.4.0",
				"MARKATA_GO_SEARCH_PAGEFIND_VERBOSE":      "true",
			},
			expectedFunc: func(c *models.Config) bool {
				if c.Extra == nil {
					return false
				}
				searchConfig, ok := c.Extra["search"].(models.SearchConfig)
				if !ok {
					return false
				}
				return searchConfig.Pagefind.AutoInstall != nil && *searchConfig.Pagefind.AutoInstall == true &&
					searchConfig.Pagefind.CacheDir == "/tmp/cache" &&
					searchConfig.Pagefind.Version == "v1.4.0" &&
					searchConfig.Pagefind.Verbose != nil && *searchConfig.Pagefind.Verbose == true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				// Clean up environment variables
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			// Create a config with default search settings
			config := &models.Config{
				Extra: map[string]interface{}{
					"search": models.NewSearchConfig(),
				},
			}

			// Apply env overrides
			err := ApplyEnvOverrides(config)
			if err != nil {
				t.Fatalf("ApplyEnvOverrides() error = %v", err)
			}

			// Check expected condition
			if !tt.expectedFunc(config) {
				t.Errorf("ApplyEnvOverrides() did not set expected values for %s", tt.name)
			}
		})
	}
}
