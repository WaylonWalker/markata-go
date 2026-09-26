package themes

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestReaderMode_LayoutKeepsPostReadable(t *testing.T) {
	components, err := ReadStatic("css/components.css")
	if err != nil {
		t.Fatal(err)
	}
	main, err := ReadStatic("css/main.css")
	if err != nil {
		t.Fatal(err)
	}

	checks := []struct {
		name, css, selector, declaration string
	}{
		{"hide pinned feed drawer", string(components), "body.reader-mode .feed-sidebar", "display: none"},
		{"hide pinned content drawer", string(components), "body.reader-mode .content-sidebar", "display: none"},
		{"hide TOC drawer", string(components), "body.reader-mode .doc-sidebar", "display: none"},
		{"remove dimming scrim", string(components), "body.reader-mode .page-wrapper::before", "display: none"},
		{"restore full page width", string(components), "body.reader-mode .page-wrapper", "max-width: var(--page-width, 1200px)"},
		{"remove pinned sidebar push", string(components), "body.reader-mode .page-wrapper > .main-content", "width: 100%"},
		{"remove empty TOC column", string(components), "body.reader-mode .content-wrapper--with-sidebar", "display: block"},
		{"keep exit hint from covering the page", string(components), "body.reader-mode::after", "inset: auto 1rem 1rem auto"},
		{"widen article measure", string(main), "body.reader-mode article.post", "max-width: 82ch"},
		{"keep padding inside mobile width", string(main), "body.reader-mode article.post", "box-sizing: border-box"},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			rule := regexp.MustCompile(regexp.QuoteMeta(check.selector) + `\s*\{([^}]*)\}`)
			match := rule.FindStringSubmatch(check.css)
			if len(match) < 2 || !strings.Contains(match[1], check.declaration) {
				t.Errorf("%s must include %q", check.selector, check.declaration)
			}
		})
	}
}

func TestReaderMode_ShortcutHelpDescribesBothContexts(t *testing.T) {
	for _, path := range []string{
		"../../templates/partials/shortcuts-modal.html",
		"default/templates/partials/shortcuts-modal.html",
	} {
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			help := strings.ReplaceAll(string(content), "\r\n", "\n")
			if !strings.Contains(help, "<kbd>s</kbd></td>\n") ||
				!strings.Contains(help, "Toggle reader mode (posts) / simple/rich view (feeds)") {
				t.Error("s shortcut help must describe both post and feed behavior")
			}
		})
	}
}
