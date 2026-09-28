package themes

import (
	"strings"
	"testing"
)

func TestConditionalCSS_ResetsNavHoverAcrossViewTransitions(t *testing.T) {
	content, err := ReadStatic("js/conditional-css.js")
	if err != nil {
		t.Fatalf("ReadStatic(conditional-css.js) error = %v", err)
	}

	js := strings.ReplaceAll(string(content), "\r\n", "\n")
	for _, needle := range []string{
		"data-nav-hover-suppressed",
		"function suppressNavHoverUntilPointerMoves()",
		".nav-entry--preview:hover:not(:focus-within) .nav-preview",
		"document.addEventListener('click'",
		"window.addEventListener('pointermove'",
		"window.addEventListener('view-transition-complete'",
		"if (navHoverSuppressed)",
	} {
		if !strings.Contains(js, needle) {
			t.Fatalf("conditional-css.js missing nav-hover reset behavior %q", needle)
		}
	}
}
