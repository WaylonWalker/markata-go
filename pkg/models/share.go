package models

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	sharePlatformCopyKey = "copy"
	shareActionCopy      = "copy"
)

// DefaultSharePlatformOrder defines the built-in share buttons in the default order.
var DefaultSharePlatformOrder = []string{"twitter", "bluesky", "linkedin", "whatsapp", "signal", "facebook", "telegram", "pinterest", "reddit", "hacker_news", "email", "copy"}

type sharePlatformDefinition struct {
	Name     string
	Icon     string
	Template string
	Action   string
}

var sharePlatformDefinitions = map[string]sharePlatformDefinition{
	"twitter": {
		Name:     "Twitter",
		Icon:     "icons/share/twitter.svg",
		Template: "https://twitter.com/intent/tweet?text={{title}}&url={{url}}",
	},
	"facebook": {
		Name:     "Facebook",
		Icon:     "icons/share/facebook.svg",
		Template: "https://www.facebook.com/sharer/sharer.php?u={{url}}",
	},
	"bluesky": {
		Name:     "Bluesky",
		Icon:     "icons/share/bluesky.svg",
		Template: "https://bsky.app/intent/compose?text={{url}}",
	},
	"linkedin": {
		Name:     "LinkedIn",
		Icon:     "icons/share/linkedin.svg",
		Template: "https://www.linkedin.com/sharing/share-offsite/?url={{url}}",
	},
	"whatsapp": {
		Name:     "WhatsApp",
		Icon:     "icons/share/whatsapp.svg",
		Template: "https://wa.me/?text={{url}}",
	},
	"signal": {
		Name:     "Signal",
		Icon:     "icons/share/signal.svg",
		Template: "https://signal.me/?text={{url}}",
	},
	"telegram": {
		Name:     "Telegram",
		Icon:     "icons/share/telegram.svg",
		Template: "https://t.me/share/url?url={{url}}",
	},
	"pinterest": {
		Name:     "Pinterest",
		Icon:     "icons/share/pinterest.svg",
		Template: "https://pinterest.com/pin/create/button/?url={{url}}",
	},
	"reddit": {
		Name:     "Reddit",
		Icon:     "icons/share/reddit.svg",
		Template: "https://reddit.com/submit?url={{url}}&title={{title}}",
	},
	"hacker_news": {
		Name:     "Hacker News",
		Icon:     "icons/share/hacker_news.svg",
		Template: "https://news.ycombinator.com/submitlink?u={{url}}&t={{title}}",
	},
	"email": {
		Name:     "Email",
		Icon:     "icons/share/email.svg",
		Template: "mailto:?subject={{title}}&body={{url}}",
	},
	sharePlatformCopyKey: {
		Name:   "Copy link",
		Icon:   "icons/share/copy.svg",
		Action: shareActionCopy,
	},
}

// ShareButton exposes data required by share templates.
type ShareButton struct {
	Key              string
	Name             string
	Icon             string
	IconIsThemeAsset bool
	Action           string
	Link             string
	CopyText         string
	AriaLabel        string
	CopyFeedback     string
}

// BuildShareButtons creates the share buttons for a specific post using the current config.
func BuildShareButtons(cfg ShareComponentConfig, baseURL, fallbackTitle string, post *Post) []ShareButton {
	if post == nil || !cfg.IsEnabled() {
		return nil
	}

	order := cfg.Platforms
	if len(order) == 0 {
		order = append([]string{}, DefaultSharePlatformOrder...)
	}

	postURL := buildPostURL(baseURL, post.Href)
	placeholders := newSharePlaceholders(post, fallbackTitle, postURL)

	buttons := make([]ShareButton, 0, len(order))
	for _, key := range order {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		def, hasBuiltin := sharePlatformDefinitions[key]
		custom, hasCustom := cfg.Custom[key]
		if !hasBuiltin && !hasCustom {
			continue
		}

		if hasCustom {
			if custom.Name != "" {
				def.Name = custom.Name
			}
			if custom.Icon != "" {
				def.Icon = custom.Icon
			}
			if custom.URL != "" {
				def.Template = custom.URL
			}
		}

		name := def.Name
		if name == "" {
			name = fallbackSharePlatformName(key)
		}

		icon := def.Icon
		iconIsThemeAsset := shouldTreatAsThemeAsset(icon)
		if icon == "" {
			icon = "icons/share/copy.svg"
			iconIsThemeAsset = true
		}

		if key == sharePlatformCopyKey || def.Action == shareActionCopy {
			buttons = append(buttons, ShareButton{
				Key:              key,
				Name:             name,
				Icon:             icon,
				IconIsThemeAsset: iconIsThemeAsset,
				Action:           shareActionCopy,
				CopyText:         postURL,
				AriaLabel:        "Copy link to clipboard",
				CopyFeedback:     "Link copied to clipboard",
			})
			continue
		}

		if def.Template == "" {
			continue
		}

		buttons = append(buttons, ShareButton{
			Key:              key,
			Name:             name,
			Icon:             icon,
			IconIsThemeAsset: iconIsThemeAsset,
			Action:           "share",
			Link:             applySharePlaceholders(def.Template, placeholders),
			AriaLabel:        fmt.Sprintf("Share on %s", name),
		})
	}

	return buttons
}

func shouldTreatAsThemeAsset(path string) bool {
	if path == "" {
		return true
	}
	lower := strings.ToLower(path)
	return !(strings.HasPrefix(lower, "http") || strings.HasPrefix(path, "/") || strings.HasPrefix(lower, "data:"))
}

func fallbackSharePlatformName(key string) string {
	words := strings.Fields(strings.ReplaceAll(key, "_", " "))
	for i := range words {
		if words[i] == "" {
			continue
		}
		if len(words[i]) == 1 {
			words[i] = strings.ToUpper(words[i])
			continue
		}
		words[i] = strings.ToUpper(words[i][:1]) + strings.ToLower(words[i][1:])
	}
	return strings.Join(words, " ")
}

func buildPostURL(base, href string) string {
	if href == "" {
		href = "/"
	}
	if base == "" {
		return href
	}
	clean := strings.TrimRight(base, "/")
	if href == "/" {
		return clean + "/"
	}
	return clean + href
}

func newSharePlaceholders(post *Post, fallbackTitle, postURL string) map[string]string {
	title := fallbackTitle
	if post.Title != nil && *post.Title != "" {
		title = *post.Title
	}
	if title == "" {
		title = post.Slug
	}
	excerpt := ""
	if post.Description != nil {
		excerpt = *post.Description
	} else if v, ok := post.Extra["excerpt"]; ok {
		if str, ok := v.(string); ok {
			excerpt = str
		}
	}

	return map[string]string{
		"title":   url.QueryEscape(title),
		"url":     url.QueryEscape(postURL),
		"excerpt": url.QueryEscape(excerpt),
	}
}

func applySharePlaceholders(template string, substitutions map[string]string) string {
	result := template
	for key, value := range substitutions {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}
