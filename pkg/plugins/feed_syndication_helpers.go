// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"path/filepath"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func getSyndicationConfig(config *lifecycle.Config) models.SyndicationConfig {
	if config != nil && config.Extra != nil {
		if defaults, ok := config.Extra["feed_defaults"].(models.FeedDefaults); ok {
			return defaults.Syndication
		}
		if defaults, ok := config.Extra["feeds.defaults"].(models.FeedDefaults); ok {
			return defaults.Syndication
		}
	}
	return models.NewFeedDefaults().Syndication
}

func getFeedsPageConfig(config *lifecycle.Config) models.FeedsPageConfig {
	if config != nil && config.Extra != nil {
		if feedsPage, ok := config.Extra["feeds_page"].(models.FeedsPageConfig); ok {
			return feedsPage
		}
	}
	return models.NewFeedsPageConfig()
}

func isRootFeed(fc *models.FeedConfig) bool {
	return fc != nil && fc.Slug == ""
}

func isArchiveFeed(fc *models.FeedConfig) bool {
	return fc != nil && fc.Slug == defaultArchivePrefix
}

func shouldGenerateFeedArchive(fc *models.FeedConfig, syndication models.SyndicationConfig) bool {
	if fc == nil {
		return false
	}
	if isRootFeed(fc) || isArchiveFeed(fc) {
		return false
	}
	if syndication.FeedArchivesDisabled || fc.ArchiveDisabled {
		return false
	}
	return fc.Formats.RSS || fc.Formats.Atom || fc.Formats.JSON
}

func feedArchiveDir(feedDir string) string {
	return filepath.Join(feedDir, defaultArchivePrefix)
}

func feedArchiveURL(slug, fileName string) string {
	cleanSlug := strings.Trim(slug, "/")
	if cleanSlug == "" {
		return "/" + defaultArchivePrefix + "/" + fileName
	}
	return "/" + cleanSlug + "/" + defaultArchivePrefix + "/" + fileName
}

func limitFeedPosts(posts []*models.Post, maxItems int) []*models.Post {
	if maxItems <= 0 || len(posts) <= maxItems {
		return posts
	}
	return posts[:maxItems]
}

func cloneFeedConfigWithPosts(fc *models.FeedConfig, posts []*models.Post) *models.FeedConfig {
	clone := *fc
	clone.Posts = posts
	return &clone
}
