// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

var stableFallbackTime = time.Unix(0, 0).UTC()

type siteMetadata struct {
	URL            string
	Title          string
	Description    string
	Author         string
	Language       string
	AuthorURL      string
	ManagingEditor string
	WebMaster      string
	Copyright      string
	LogoURL        string
	Authors        map[string]models.Author
}

func getSiteMetadata(config *lifecycle.Config) siteMetadata {
	meta := siteMetadata{
		URL:            getSiteURL(config),
		Title:          getSiteTitle(config),
		Description:    getSiteDescription(config),
		Author:         getSiteAuthor(config),
		Language:       getSiteLanguage(config),
		AuthorURL:      getSiteAuthorURL(config),
		ManagingEditor: getSiteManagingEditor(config),
		WebMaster:      getSiteWebMaster(config),
		Copyright:      getSiteCopyright(config),
	}

	if modelsConfig, ok := config.Extra["models_config"].(*models.Config); ok && modelsConfig != nil {
		meta.LogoURL = modelsConfig.SEO.LogoURL
		meta.Authors = modelsConfig.Authors.Authors
	}

	if meta.Author == "" {
		meta.Author = meta.Title
	}

	return meta
}

func feedResolvedTitle(feed *lifecycle.Feed, meta siteMetadata) string {
	if feed != nil && feed.Title != "" {
		return feed.Title
	}
	if meta.Title != "" {
		return meta.Title
	}
	return "Feed"
}

func feedResolvedDescription(feed *lifecycle.Feed, meta siteMetadata) string {
	if feed != nil && feed.Description != "" {
		return feed.Description
	}
	return meta.Description
}

func isArchiveFeedPath(feedPath string) bool {
	clean := strings.Trim(strings.TrimSpace(feedPath), "/")
	return clean == defaultArchivePrefix || strings.HasSuffix(clean, "/"+defaultArchivePrefix)
}

func feedURLForFormat(siteURL, feedPath, fileName string) string {
	cleanSiteURL := strings.TrimSuffix(siteURL, "/")
	cleanPath := strings.Trim(feedPath, "/")
	if cleanPath == "" {
		return cleanSiteURL + "/" + fileName
	}
	return cleanSiteURL + "/" + cleanPath + "/" + fileName
}

func feedHomePageURL(siteURL, feedPath string) string {
	cleanSiteURL := strings.TrimSuffix(siteURL, "/")
	cleanPath := strings.Trim(feedPath, "/")
	if cleanPath == "" || cleanPath == defaultArchivePrefix {
		return cleanSiteURL + "/"
	}
	if strings.HasSuffix(cleanPath, "/"+defaultArchivePrefix) {
		cleanPath = strings.TrimSuffix(cleanPath, "/"+defaultArchivePrefix)
	}
	if cleanPath == "" {
		return cleanSiteURL + "/"
	}
	return cleanSiteURL + "/" + cleanPath + "/"
}

func latestFeedTime(posts []*models.Post) time.Time {
	latest := stableFallbackTime
	for _, post := range posts {
		if post == nil || post.Date == nil {
			continue
		}
		if post.Date.After(latest) {
			latest = *post.Date
		}
	}
	return latest.UTC()
}

func postUpdatedTime(post *models.Post, fallback time.Time) time.Time {
	if post != nil && post.Date != nil {
		return post.Date.UTC()
	}
	return fallback.UTC()
}

func firstAuthorForPost(post *models.Post, meta siteMetadata) *models.Author {
	if post == nil || len(meta.Authors) == 0 {
		return nil
	}
	for _, id := range post.GetAuthors() {
		author, ok := meta.Authors[id]
		if ok {
			return &author
		}
	}
	return nil
}

func archiveCurrentFeedPath(feedPath string) string {
	clean := strings.Trim(feedPath, "/")
	if clean == defaultArchivePrefix {
		return ""
	}
	return strings.Trim(strings.TrimSuffix(clean, "/"+defaultArchivePrefix), "/")
}

func archiveTitleSuffix(title string) string {
	if title == "" || strings.HasSuffix(title, " Archive") {
		return title
	}
	return title + " Archive"
}

func feedArchiveCurrentURL(siteURL, feedPath, fileName string) string {
	currentPath := archiveCurrentFeedPath(feedPath)
	return feedURLForFormat(siteURL, currentPath, fileName)
}
