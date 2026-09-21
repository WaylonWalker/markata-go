package themes

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

var (
	cssCommentRe = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssStringRe  = regexp.MustCompile(`"[^"\n]*"|'[^'\n]*'`)
)

// stripCSSNoise removes comments and string literals so brace counting only
// sees structural braces.
func stripCSSNoise(css string) string {
	css = cssCommentRe.ReplaceAllString(css, "")
	return cssStringRe.ReplaceAllString(css, `""`)
}

// TestCSSLayerBlocksStayBalanced guards against a stray brace closing a
// top-level @layer block early. When that happens every rule after the stray
// brace becomes unlayered and outranks all layered rules across the theme,
// which silently breaks the cascade for unrelated components.
func TestCSSLayerBlocksStayBalanced(t *testing.T) {
	static := DefaultStatic()
	err := fs.WalkDir(static, "css", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".css") {
			return nil
		}
		raw, readErr := fs.ReadFile(static, p)
		if readErr != nil {
			return readErr
		}
		css := stripCSSNoise(string(raw))

		t.Run(p, func(t *testing.T) {
			depth := 0
			line := 1
			topLevelLayerOpenedAt := 0
			for _, r := range css {
				switch r {
				case '\n':
					line++
				case '{':
					depth++
				case '}':
					depth--
					if depth < 0 {
						t.Fatalf("%s:%d: unmatched closing brace", p, line)
					}
					if depth == 0 && topLevelLayerOpenedAt > 0 {
						topLevelLayerOpenedAt = 0
					}
				}
			}
			if depth != 0 {
				t.Fatalf("%s: unbalanced braces (depth %d at EOF)", p, depth)
			}

			// Any file that opens a top-level `@layer name {` block must keep
			// all of its rules inside layer blocks.
			if !regexp.MustCompile(`(?m)^@layer\s+[a-z-]+\s*\{`).MatchString(css) {
				return
			}
			depth = 0
			line = 1
			inLayerBlock := false
			for i := 0; i < len(css); i++ {
				c := css[i]
				switch c {
				case '\n':
					line++
				case '{':
					if depth == 0 {
						stmtStart := strings.LastIndexAny(css[:i], "};")
						stmt := strings.TrimSpace(css[stmtStart+1 : i])
						inLayerBlock = strings.HasPrefix(stmt, "@layer")
						if !inLayerBlock {
							t.Errorf("%s:%d: rule %q is outside any @layer block", p, line, firstLine(stmt))
						}
					}
					depth++
				case '}':
					depth--
				}
			}
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk css: %v", err)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}
