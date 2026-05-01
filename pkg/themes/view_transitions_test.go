package themes

import (
	"strings"
	"testing"
)

func TestViewTransitions_SyncHeadScriptsRemovesStaleExternalScripts(t *testing.T) {
	content, err := ReadStatic("js/view-transitions.js")
	if err != nil {
		t.Fatalf("ReadStatic(view-transitions.js) error = %v", err)
	}

	js := string(content)
	for _, needle := range []string{
		"function syncHeadScripts(newDoc)",
		"function shouldRemoveManagedHeadScript(node)",
		"return true;",
		"data-markata-persist",
		"same managed\n   * head state as a full reload",
	} {
		if !strings.Contains(js, needle) {
			t.Fatalf("view-transitions.js missing %q", needle)
		}
	}

	if strings.Contains(js, "return !node.hasAttribute('src');") {
		t.Fatal("view-transitions.js still keeps stale external managed scripts")
	}
}
