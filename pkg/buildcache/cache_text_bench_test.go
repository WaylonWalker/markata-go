package buildcache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A package-level sink forces the baseline byte-to-string copy to escape.
var cacheTextBenchmarkResult string

func BenchmarkReadCacheTextFile(b *testing.B) {
	for _, size := range []int{1024, 64 * 1024, 1024 * 1024} {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			path := filepath.Join(b.TempDir(), "page.html")
			if err := os.WriteFile(path, []byte(strings.Repeat("x", size)), 0o600); err != nil {
				b.Fatal(err)
			}
			// Prime the filesystem cache before either timed implementation.
			if _, err := readCacheTextFile(path); err != nil {
				b.Fatal(err)
			}
			b.Run("ReadFileString", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(size))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					data, err := os.ReadFile(path)
					if err != nil {
						b.Fatal(err)
					}
					cacheTextBenchmarkResult = string(data)
				}
			})
			b.Run("BuilderPooled", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(size))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					text, err := readCacheTextFile(path)
					if err != nil {
						b.Fatal(err)
					}
					cacheTextBenchmarkResult = text
				}
			})
		})
	}
}
