package buildcache

import (
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"unsafe"
)

func TestReadCacheTextFile(t *testing.T) {
	for _, content := range []string{"", "<p>hello</p>\n", "NUL\x00invalid\xff\xfeUTF-8", strings.Repeat("large\x00\xff", 20000)} {
		t.Run(strconv.Itoa(len(content))+"B", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "page.html")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := readCacheTextFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != content {
				t.Fatalf("restored bytes differ: got length %d, want %d", len(got), len(content))
			}
		})
	}
}

func TestReadCacheTextFileErrors(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{filepath.Join(dir, "missing.html"), dir} {
		got, err := readCacheTextFile(path)
		if err == nil || got != "" {
			t.Errorf("readCacheTextFile(%q) = %q, %v; want empty string and error", path, got, err)
		}
	}
	t.Run("permission denied", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows file modes do not deny read access")
		}
		if os.Geteuid() == 0 {
			t.Skip("root can read permission-denied fixtures")
		}
		path := filepath.Join(dir, "unreadable.html")
		if err := os.WriteFile(path, []byte("must not restore"), 0o000); err != nil {
			t.Fatal(err)
		}
		got, err := readCacheTextFile(path)
		if got != "" || !errors.Is(err, os.ErrPermission) {
			t.Fatalf("permission failure = %q, %v; want empty string and permission error", got, err)
		}
	})
}

func TestReadCacheTextImmutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "page.html")
	original := strings.Repeat("original\x00\xff", 10000)
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := readCacheTextFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		next := strings.Repeat("replacement", 20000+i)
		if err := os.WriteFile(path, []byte(next), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := readCacheTextFile(path)
		if err != nil || got != next {
			t.Fatalf("subsequent read %d: length %d, error %v", i, len(got), err)
		}
		if first != original {
			t.Fatal("previous result changed after transfer buffer reuse")
		}
	}
}

func TestConcurrentHTMLDiskRestoration(t *testing.T) {
	const workers, reads = 6, 16
	dir := t.TempDir()
	cache := New("")
	paths := make([]string, workers)
	contents := make([]string, workers)
	results := make([][]string, workers)
	for worker := range workers {
		label := strconv.Itoa(worker)
		paths[worker] = filepath.Join(dir, label+".html")
		// Distinct bytes and lengths exercise multiple transfers per file.
		contents[worker] = strings.Repeat(label+"\x00\xff\xfe\xc0 HTML\n", 6000+worker*1000)
		if err := os.WriteFile(paths[worker], []byte(contents[worker]), 0o600); err != nil {
			t.Fatal(err)
		}
		cache.Posts[paths[worker]] = &PostCache{FullHTMLPath: paths[worker]}
	}

	var done sync.WaitGroup
	start := make(chan struct{})
	for worker := range workers {
		done.Add(1)
		go func() {
			defer done.Done()
			<-start
			path := paths[worker]
			results[worker] = make([]string, 0, reads*2)
			for read := range reads {
				text, err := readCacheTextFile(path)
				if err != nil || text != contents[worker] {
					t.Errorf("worker %d read %d: helper bytes differ or error: %v", worker, read, err)
					return
				}
				results[worker] = append(results[worker], text)

				// Each worker owns a distinct key. Force the actual getter's
				// disk fallback on every iteration while other keys read.
				cache.fullHTMLMemory.Delete(path)
				html := cache.GetCachedFullHTML(path)
				if html != contents[worker] {
					t.Errorf("worker %d read %d: full HTML getter bytes differ", worker, read)
					return
				}
				results[worker] = append(results[worker], html)
			}
		}()
	}
	close(start)
	done.Wait()

	// Retain every result until all goroutines finish reusing pooled buffers.
	for worker, stringsRead := range results {
		if len(stringsRead) != reads*2 {
			t.Errorf("worker %d returned %d strings, want %d", worker, len(stringsRead), reads*2)
		}
		for read, text := range stringsRead {
			if text != contents[worker] {
				t.Errorf("worker %d retained result %d changed after concurrent reads", worker, read)
			}
		}
	}
}

func TestConcurrentFullHTMLSharedBacking(t *testing.T) {
	const workers = 16
	path := filepath.Join(t.TempDir(), "shared.html")
	content := strings.Repeat("shared full page\x00\xff", 200000)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cache := New("")
	for i := range workers {
		cache.Posts[strconv.Itoa(i)] = &PostCache{FullHTMLPath: path}
	}
	results := make([]string, workers)
	start := make(chan struct{})
	var done sync.WaitGroup
	for i := range workers {
		done.Add(1)
		go func() {
			defer done.Done()
			<-start
			results[i] = cache.GetCachedFullHTML(strconv.Itoa(i))
		}()
	}
	close(start)
	done.Wait()
	for i, html := range results {
		if html != content {
			t.Fatalf("worker %d restored different bytes", i)
		}
		if unsafe.StringData(html) != unsafe.StringData(results[0]) {
			t.Fatalf("worker %d retained a duplicate backing allocation for the shared cache path", i)
		}
	}
}

func TestCacheTextCapacity(t *testing.T) {
	for _, tc := range []struct {
		name    string
		size    int64
		regular bool
		want    int
	}{
		{"empty", 0, true, 0},
		{"unknown", -1, true, 0},
		{"nonregular", 1024, false, 0},
		{"small", 1024, true, 1024},
		{"bounded", maxCacheTextPreallocation + 1, true, maxCacheTextPreallocation},
		{"untrusted", math.MaxInt64, true, maxCacheTextPreallocation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := cacheTextCapacity(tc.size, tc.regular); got != tc.want {
				t.Errorf("capacity = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestReadCacheTextSizeIsOnlyHint(t *testing.T) {
	content := strings.Repeat("bytes\x00\xff", 10000)
	for _, size := range []int64{-1, 0, 1, int64(len(content)) + 1000, math.MaxInt64} {
		got, err := readCacheText(strings.NewReader(content), size, true)
		if err != nil || got != content {
			t.Fatalf("size hint %d: length %d, error %v", size, len(got), err)
		}
	}
	// The preallocation bound must not become a content-size limit.
	large := strings.Repeat("x", maxCacheTextPreallocation+1)
	got, err := readCacheText(strings.NewReader(large), int64(len(large)), true)
	if err != nil || got != large {
		t.Fatalf("content above allocation hint bound: length %d, error %v", len(got), err)
	}
}

type failingCacheTextReader struct {
	err error
}

func (r failingCacheTextReader) Read(p []byte) (int, error) {
	return copy(p, "partial HTML"), r.err
}

func TestReadCacheTextReadFailure(t *testing.T) {
	wantErr := errors.New("injected read failure")
	got, err := readCacheText(failingCacheTextReader{wantErr}, 100, true)
	if got != "" || !errors.Is(err, wantErr) {
		t.Fatalf("read failure = %q, %v; want empty string and original error", got, err)
	}
}

type writerToCacheTextReader struct {
	io.Reader
	called bool
}

func (r *writerToCacheTextReader) WriteTo(io.Writer) (int64, error) {
	r.called = true
	return 0, errors.New("WriterTo must not bypass pooled buffer")
}

func TestReadCacheTextHidesWriterTo(t *testing.T) {
	reader := &writerToCacheTextReader{Reader: strings.NewReader("complete HTML")}
	got, err := readCacheText(reader, 13, true)
	if err != nil || got != "complete HTML" || reader.called {
		t.Fatalf("read = %q, %v; WriterTo called = %v", got, err, reader.called)
	}
}

func TestHTMLDiskCacheRestoration(t *testing.T) {
	for _, kind := range []string{"full", "article"} {
		t.Run(kind, func(t *testing.T) {
			cache := New("")
			path := filepath.Join(t.TempDir(), "cached.html")
			content := "<article>exact\x00\xff\xfe bytes</article>\n"
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			const source = "post.md"
			cache.Posts[source] = &PostCache{
				FullHTMLPath: path, ArticleHTMLPath: path, ContentHash: "current",
			}
			get := func() string { return cache.GetCachedFullHTML(source) }
			if kind == "article" {
				get = func() string { return cache.GetCachedArticleHTML(source, "current") }
				if got := cache.GetCachedArticleHTML(source, "stale"); got != "" {
					t.Fatal("stale article hash restored HTML from disk")
				}
			}
			if got := get(); got != content {
				t.Fatal("disk restoration did not preserve exact bytes")
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if got := get(); got != content {
				t.Fatal("memory hit accessed removed disk cache")
			}
			if kind == "article" && cache.GetCachedArticleHTML(source, "stale") != "" {
				t.Fatal("stale article hash bypassed memory freshness gate")
			}
		})
	}
}

func TestHTMLDiskCacheMiss(t *testing.T) {
	dir := t.TempDir()
	paths := []string{"", filepath.Join(dir, "missing.html"), dir}
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		path := filepath.Join(dir, "unreadable.html")
		if err := os.WriteFile(path, []byte("must not restore"), 0o000); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	for _, path := range paths {
		cache := New("")
		cache.Posts["post.md"] = &PostCache{
			FullHTMLPath: path, ArticleHTMLPath: path, ContentHash: "current",
		}
		if got := cache.GetCachedFullHTML("post.md"); got != "" {
			t.Errorf("full HTML cache miss = %q", got)
		}
		if got := cache.GetCachedArticleHTML("post.md", "current"); got != "" {
			t.Errorf("article HTML cache miss = %q", got)
		}
		if _, ok := cache.fullHTMLMemory.Load(path); ok {
			t.Error("failed full HTML read stored a memory entry")
		}
		if _, ok := cache.articleHTMLMemory.Load(path); ok {
			t.Error("failed article HTML read stored a memory entry")
		}
	}
}

func TestHTMLEmptyDiskCache(t *testing.T) {
	cache := New("")
	path := filepath.Join(t.TempDir(), "empty.html")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	cache.Posts["post.md"] = &PostCache{
		FullHTMLPath: path, ArticleHTMLPath: path, ContentHash: "current",
	}
	if cache.GetCachedFullHTML("post.md") != "" || cache.GetCachedArticleHTML("post.md", "current") != "" {
		t.Fatal("empty cache file restored nonempty HTML")
	}
	// Empty successful reads are distinguishable internally from failed reads.
	for _, memory := range []*sync.Map{&cache.fullHTMLMemory, &cache.articleHTMLMemory} {
		got, ok := memory.Load(path)
		if !ok || got != "" {
			t.Fatalf("empty HTML memory entry = %v, %v; want empty string and true", got, ok)
		}
	}
}

func TestHTMLDiskCacheAllocationBudget(t *testing.T) {
	// Bytes, rather than allocation counts, detect a second body-sized buffer.
	// The margin allows filesystem bookkeeping and race-mode pool misses, but
	// is smaller than another body for both representative page sizes.
	for _, size := range []int{64 * 1024, 1024 * 1024} {
		for _, kind := range []string{"full", "article"} {
			t.Run(kind+"/"+strconv.Itoa(size), func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "page.html")
				if err := os.WriteFile(path, []byte(strings.Repeat("x", size)), 0o600); err != nil {
					t.Fatal(err)
				}
				cache := New("")
				cache.Posts["post.md"] = &PostCache{
					FullHTMLPath: path, ArticleHTMLPath: path, ContentHash: "current",
				}
				get := func() string {
					cache.fullHTMLMemory.Delete(path)
					return cache.GetCachedFullHTML("post.md")
				}
				if kind == "article" {
					get = func() string {
						cache.articleHTMLMemory.Delete(path)
						return cache.GetCachedArticleHTML("post.md", "current")
					}
				}
				if len(get()) != size {
					t.Fatal("disk cache priming failed")
				}
				result := testing.Benchmark(func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						cacheTextBenchmarkResult = get()
					}
				})
				budget := int64(size + size/2 + 4096)
				t.Logf("disk restoration: %d B/op; budget %d B/op", result.AllocedBytesPerOp(), budget)
				if result.AllocedBytesPerOp() > budget {
					t.Fatalf("disk restoration allocated %d B/op, exceeds %d B/op: possible body-sized copy",
						result.AllocedBytesPerOp(), budget)
				}
			})
		}
	}
}
