package buildcache

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	cacheTextBufferSize = 32 * 1024
	// Metadata is only a hint. Do not eagerly allocate huge sparse file sizes.
	maxCacheTextPreallocation = 8 * 1024 * 1024
)

var cacheTextBuffers = sync.Pool{
	New: func() any { return new([cacheTextBufferSize]byte) },
}

// cacheTextReader hides WriterTo (notably os.File.WriteTo), so CopyBuffer uses
// our bounded transfer buffer rather than allocating its own.
type cacheTextReader struct {
	io.Reader
}

// readCacheTextFile restores exact bytes into string-owned storage. Errors,
// including close errors, never return a partially restored cache entry.
func readCacheTextFile(path string) (text string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening HTML cache: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			text = ""
			err = fmt.Errorf("closing HTML cache: %w", closeErr)
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("statting HTML cache: %w", err)
	}
	return readCacheText(file, info.Size(), info.Mode().IsRegular())
}

// cacheTextCapacity bounds metadata-driven allocation, not the content read.
func cacheTextCapacity(size int64, regular bool) int {
	if !regular || size <= 0 {
		return 0
	}
	maxInt := int64(^uint(0) >> 1)
	if size > maxInt || size > maxCacheTextPreallocation {
		return maxCacheTextPreallocation
	}
	return int(size)
}

func readCacheText(reader io.Reader, size int64, regular bool) (string, error) {
	var builder strings.Builder
	builder.Grow(cacheTextCapacity(size, regular))

	buffer, ok := cacheTextBuffers.Get().(*[cacheTextBufferSize]byte)
	if !ok || buffer == nil {
		panic("buildcache: invalid internal text buffer")
	}
	defer cacheTextBuffers.Put(buffer)

	if _, err := io.CopyBuffer(&builder, cacheTextReader{reader}, buffer[:]); err != nil {
		return "", fmt.Errorf("reading HTML cache: %w", err)
	}
	return builder.String(), nil
}
