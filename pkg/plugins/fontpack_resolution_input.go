package plugins

import "html"

// fontpackResolutionInput reduces a full rendered site to the only content
// dependency used by tier selection: the unique visible rune set. Escaping the
// compact string keeps characters such as '<' and '&' visible when the
// fontpacks resolver runs its existing HTML text extraction.
func fontpackResolutionInput(renderedHTML string) string {
	return html.EscapeString(fontpackCoverageSignature(renderedHTML))
}
