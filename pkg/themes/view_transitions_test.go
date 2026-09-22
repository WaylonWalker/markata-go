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
		"const RUNTIME_HTML_ATTRIBUTES = new Set(['data-theme', 'data-text-size']);",
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

func TestViewTransitions_SidebarClosePreservesKeyboardFocus(t *testing.T) {
	content, err := ReadStatic("js/view-transitions.js")
	if err != nil {
		t.Fatalf("ReadStatic(view-transitions.js) error = %v", err)
	}

	js := strings.ReplaceAll(string(content), "\r\n", "\n")
	for _, needle := range []string{
		"active !== btn",
		"btn.focus();",
		"window._sidebarEscapeBound",
		"e.key !== 'Escape' || window.innerWidth < 1201",
		"aria-expanded",
	} {
		if !strings.Contains(js, needle) {
			t.Fatalf("view-transitions.js missing keyboard drawer behavior %q", needle)
		}
	}
	if strings.Contains(js, "active.blur()") {
		t.Fatal("sidebar closing must not discard keyboard focus")
	}
}

func TestSidebarTemplatesExposeAccessibleToggleState(t *testing.T) {
	for _, name := range []string{
		"components/feed_sidebar.html",
		"components/doc_sidebar.html",
	} {
		t.Run(name, func(t *testing.T) {
			content, err := ReadTemplate(name)
			if err != nil {
				t.Fatalf("ReadTemplate(%q) error = %v", name, err)
			}
			template := string(content)
			for _, needle := range []string{
				`class="sidebar-toggle"`,
				`aria-expanded="false"`,
				`data-sidebar-label=`,
			} {
				if !strings.Contains(template, needle) {
					t.Fatalf("%s missing accessible drawer toggle attribute %q", name, needle)
				}
			}
		})
	}
}

func TestReading_SidenoteCopiesAreHiddenFromAssistiveTechnology(t *testing.T) {
	content, err := ReadStatic("js/reading.js")
	if err != nil {
		t.Fatalf("ReadStatic(reading.js) error = %v", err)
	}

	js := strings.ReplaceAll(string(content), "\r\n", "\n")
	for _, needle := range []string{
		"footnotes.setAttribute('role', 'doc-endnotes')",
		"note.setAttribute('aria-hidden', 'true')",
		"note.setAttribute('inert', '')",
		"note.setAttribute('data-sidenote-for', target.id)",
		"control.removeAttribute('href')",
		"control.setAttribute('disabled', '')",
		"ref.addEventListener('pointerdown', markPointerActivation)",
		"if (!pointerActivation) return;",
		"ref.removeEventListener('pointerdown', markPointerActivation)",
		"ref.removeEventListener('click', toggle)",
		"clearTimeout(pointerActivationTimer)",
	} {
		if !strings.Contains(js, needle) {
			t.Fatalf("reading.js missing accessible sidenote behavior %q", needle)
		}
	}
	for _, needle := range []string{
		"ref.setAttribute('aria-controls'",
		"ref.setAttribute('aria-expanded'",
	} {
		if strings.Contains(js, needle) {
			t.Fatalf("sidenote reference must not expose misleading disclosure state: %q", needle)
		}
	}
}

func TestNavigation_PreservesReaderQueryForKeyboardFallbacks(t *testing.T) {
	content, err := ReadStatic("js/navigation-shortcuts.js")
	if err != nil {
		t.Fatalf("ReadStatic(navigation-shortcuts.js) error = %v", err)
	}

	js := strings.ReplaceAll(string(content), "\r\n", "\n")
	for _, needle := range []string{
		"function preserveReaderModeURL(url)",
		"targetURL.searchParams.set('reader', '1')",
		"window.open(preserveReaderModeURL(link.href)",
		"window.location.href = preserveReaderModeURL(url)",
	} {
		if !strings.Contains(js, needle) {
			t.Fatalf("navigation-shortcuts.js missing reader-mode propagation %q", needle)
		}
	}
}

func TestViewTransitions_PreservesReaderQueryAcrossNavigation(t *testing.T) {
	content, err := ReadStatic("js/view-transitions.js")
	if err != nil {
		t.Fatalf("ReadStatic(view-transitions.js) error = %v", err)
	}

	js := strings.ReplaceAll(string(content), "\r\n", "\n")
	for _, needle := range []string{
		"function preserveNavigationContext(rawUrl, triggerElement)",
		"document.body.classList.contains('reader-mode')",
		"targetURL.searchParams.set('reader', '1')",
		"preserveNavigationContext(url, navOptions.triggerElement)",
	} {
		if !strings.Contains(js, needle) {
			t.Fatalf("view-transitions.js missing reader-mode propagation %q", needle)
		}
	}
}
