package plugins

import (
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

// appendPostDependencyCandidates records the normalized forms that can later
// resolve to a post slug. Keeping candidates for unresolved links lets a new
// post invalidate the source when it becomes resolvable.
func appendPostDependencyCandidates(dependencies *[]string, slug string) {
	if dependencies == nil {
		return
	}
	slug = strings.TrimSpace(slug)
	if fragment := strings.IndexByte(slug, '#'); fragment >= 0 {
		slug = slug[:fragment]
	}
	if slug == "" {
		return
	}
	candidates := []string{strings.ToLower(slug), models.Slugify(slug)}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		seen := false
		for _, existing := range *dependencies {
			if existing == candidate {
				seen = true
				break
			}
		}
		if !seen {
			*dependencies = append(*dependencies, candidate)
		}
	}
}
