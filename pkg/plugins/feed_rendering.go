package plugins

import "github.com/WaylonWalker/markata-go/pkg/models"

func filterFeedOutputPosts(posts []*models.Post, includePrivate bool) []*models.Post {
	renderable := make([]*models.Post, 0, len(posts))
	for _, post := range posts {
		if post == nil || post.Skip || post.Draft || !post.Published {
			continue
		}
		if post.Private && !includePrivate {
			continue
		}
		renderable = append(renderable, post)
	}
	return renderable
}
