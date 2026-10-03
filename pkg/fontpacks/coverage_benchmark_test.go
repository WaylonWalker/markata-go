package fontpacks

import (
	"strings"
	"testing"
)

// BenchmarkCoverageMultiPack bounds the site fixture to a few MiB. Construction
// is outside timing; collection and string resolution scale with bytes, whereas
// precollected resolution should depend only on distinct runes and pack count.
func BenchmarkCoverageMultiPack(b *testing.B) {
	source, err := BuiltinSource()
	if err != nil {
		b.Fatal(err)
	}
	names := SortedKeys(source.Catalog.FontPacks)
	article := `<article><h1>A realistic notebook &amp; typography</h1><p>Repeated prose, café, Ā and Ω; code and emoji 😀.</p><code>func example() { return 42 }</code><script>ignored 中文</script></article>` + "\n"
	for _, repeats := range []struct {
		name  string
		count int
	}{{"one-article", 1}, {"repeated-site", 8192}} {
		text := strings.Repeat(article, repeats.count)
		coverage := coverageFromString(text)
		b.Run(repeats.name, func(b *testing.B) {
			b.Run("collect", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(text)))
				for b.Loop() {
					if _, err := CollectCoverage(strings.NewReader(text)); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("resolve-string", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := source.Catalog.ResolveManyFSWithOptions(names, source.FS, source.Root, text, ResolveOptions{}); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("resolve-coverage", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := source.Catalog.ResolveManyFSWithCoverage(names, source.FS, source.Root, coverage, ResolveOptions{}); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
