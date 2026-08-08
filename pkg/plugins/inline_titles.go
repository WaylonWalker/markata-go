package plugins

import (
	"fmt"
	"html"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// InlineTitlesPlugin derives safe rich and plain-text representations from
// authored title source after auto-title inference and before metadata
// plugins run.
type InlineTitlesPlugin struct{}

// NewInlineTitlesPlugin creates a title representation plugin.
func NewInlineTitlesPlugin() *InlineTitlesPlugin { return &InlineTitlesPlugin{} }

// Name returns the unique plugin name.
func (p *InlineTitlesPlugin) Name() string { return "inline_titles" }

// Priority places title rendering immediately after auto-title inference.
func (p *InlineTitlesPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageTransform {
		return lifecycle.PriorityFirst + 1
	}
	return lifecycle.PriorityDefault
}

// Transform populates TitleHTML and TitleText without changing Title, which
// remains the authored/source title for compatibility.
func (p *InlineTitlesPlugin) Transform(m *lifecycle.Manager) error {
	value, ok := m.Cache().Get(CacheKeyInlineRenderer)
	if !ok {
		renderer := NewRenderMarkdownPlugin()
		if err := renderer.Configure(m); err != nil {
			return fmt.Errorf("configure inline title renderer: %w", err)
		}
		value, _ = m.Cache().Get(CacheKeyInlineRenderer)
	}
	renderInline, ok := value.(InlineRenderFunc)
	if !ok {
		return fmt.Errorf("inline title renderer is unavailable")
	}
	posts := m.FilterPosts(func(post *models.Post) bool {
		return !post.Skip && post.Title != nil
	})
	return m.ProcessPostsSliceConcurrently(posts, func(post *models.Post) error {
		result, err := renderInline(*post.Title)
		if err != nil {
			return err
		}
		post.TitleHTML = result.HTML
		post.TitleText = result.Text
		post.TitleTextDerived = true
		if post.Slug == "" && !post.Has("_slug_explicit") {
			post.GenerateSlugWithMode(configuredSlugMode(m.Config(), post.Path))
			post.GenerateHref()
		}
		if cache := GetBuildCache(m); cache != nil {
			cache.SetPostSlug(post.Path, post.Slug)
			feedChanged, tagChanged, gardenChanged := cache.UpdatePostSemanticHashes(
				post.Path,
				computePostFeedItemHash(post),
				computePostTagIndexHash(post),
				computePostGardenHash(post),
			)
			if feedChanged {
				cache.MarkFeedSlugChanged(post.Slug)
			}
			if tagChanged {
				cache.MarkSlugChanged(post.Slug)
			}
			if gardenChanged {
				cache.MarkSlugChanged(post.Slug)
			}
		}
		return nil
	})
}

// PopulateTitleFallback provides a safe representation for manually-created
// posts when the normal lifecycle renderer is not installed.
func PopulateTitleFallback(post *models.Post) {
	if post == nil || post.Title == nil || post.TitleTextDerived {
		return
	}
	post.TitleText = html.UnescapeString(*post.Title)
	post.TitleHTML = html.EscapeString(post.TitleText)
	post.TitleTextDerived = true
}

var (
	_ lifecycle.Plugin          = (*InlineTitlesPlugin)(nil)
	_ lifecycle.TransformPlugin = (*InlineTitlesPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*InlineTitlesPlugin)(nil)
)
