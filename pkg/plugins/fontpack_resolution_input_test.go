package plugins

import "testing"

func TestFontpackResolutionInputPreservesVisibleCoverage(t *testing.T) {
	source := `<main>Alpha &amp; beta &lt;tag&gt; é Ж 字` +
		`<script>script-only-Ω</script><style>.hidden{content:"δ"}</style>` +
		`<strong>Alpha</strong></main>`

	want := fontpackCoverageSignature(source)
	got := fontpackCoverageSignature(fontpackResolutionInput(source))
	if got != want {
		t.Fatalf("compact resolver coverage = %q, want %q", got, want)
	}
}

func TestFontpackResolutionInputShrinksRepeatedRenderedText(t *testing.T) {
	source := `<p>`
	for range 1000 {
		source += `repeated prose 123 &amp; repeated prose 123 `
	}
	source += `</p>`

	got := fontpackResolutionInput(source)
	if len(got) >= len(source)/10 {
		t.Fatalf("compact resolver input length = %d, source length = %d", len(got), len(source))
	}
}
