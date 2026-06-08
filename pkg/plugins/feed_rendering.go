package plugins

import "github.com/WaylonWalker/markata-go/pkg/models"

func filterFeedPagePosts(posts []*models.Post, includePrivate bool) []*models.Post {
	visible := make([]*models.Post, 0, len(posts))
	for _, post := range posts {
		if post == nil || post.Skip || post.Draft {
			continue
		}
		if post.Private && !includePrivate {
			continue
		}
		visible = append(visible, post)
	}
	return visible
}

func filterFeedOutputPosts(posts []*models.Post, includePrivate bool) []*models.Post {
	renderable := make([]*models.Post, 0, len(posts))
	for _, post := range filterFeedPagePosts(posts, includePrivate) {
		if post.Content == "" && post.ArticleHTML == "" {
			continue
		}
		renderable = append(renderable, post)
	}
	return renderable
}
