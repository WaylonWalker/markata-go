package plugins

import "github.com/WaylonWalker/markata-go/pkg/models"

func simpleHTMLViewEnabled(fc *models.FeedConfig) bool {
	return fc != nil && fc.Formats.SimpleHTML && fc.HasView(models.FeedViewSimple)
}
