package plugins

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestSimpleHTMLViewEnabledRequiresFormatAndView(t *testing.T) {
	fc := &models.FeedConfig{Formats: models.FeedFormats{SimpleHTML: true}, Views: []string{models.FeedViewDefault}}
	if simpleHTMLViewEnabled(fc) {
		t.Fatal("simple output enabled after view opt-out")
	}
	fc.Views = []string{models.FeedViewDefault, models.FeedViewSimple}
	if !simpleHTMLViewEnabled(fc) {
		t.Fatal("simple output should be enabled")
	}
}

func TestSidebarVariantsHideDisabledSimpleView(t *testing.T) {
	fc := &models.FeedConfig{Slug: "blog", Views: []string{models.FeedViewDefault}, Formats: models.FeedFormats{SimpleHTML: true}}
	variants := buildSidebarVariants(fc, models.SyndicationConfig{}, models.PostFormatsConfig{})
	for _, variant := range variants {
		if variant.Key == "simple" {
			t.Fatal("sidebar exposed disabled simple view")
		}
	}
}
