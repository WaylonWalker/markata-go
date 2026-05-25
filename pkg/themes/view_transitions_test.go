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

	js := strings.ReplaceAll(string(content), "\r\n", "\n")
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

func TestViewTransitions_PreservesRuntimeStateAndBoundsPrefetch(t *testing.T) {
	content, err := ReadStatic("js/view-transitions.js")
	if err != nil {
		t.Fatalf("ReadStatic(view-transitions.js) error = %v", err)
	}

	js := strings.ReplaceAll(string(content), "\r\n", "\n")
	for _, needle := range []string{
		"const MAX_PREFETCHED_DOCUMENTS = 8;",
		"const RUNTIME_HTML_ATTRIBUTES = new Set(['data-theme']);",
		"const RUNTIME_HTML_CLASS_NAMES = new Set(['dark']);",
		"'data-shared-transition-'",
		"'data-post-transition-'",
		"function reexecuteInlineModuleScripts()",
		"function hydrateCriticalLayoutScripts()",
		"updateDocument(newDoc, metrics, { reinitialize: false, hydrateCritical: true });",
		"hydrateCritical: true",
		"skipCritical: true",
		"prefetchedDocuments.size >= MAX_PREFETCHED_DOCUMENTS",
	} {
		if !strings.Contains(js, needle) {
			t.Fatalf("view-transitions.js missing %q", needle)
		}
	}
}
