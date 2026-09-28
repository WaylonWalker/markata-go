//go:build linux

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanPublishKeepsHTMLAndPaletteConsistent(t *testing.T) {
	requireAtomicDirectoryExchange(t)

	root := t.TempDir()
	finalOutput := filepath.Join(root, "output")
	writePublishGeneration(t, finalOutput, 0)

	server := httptest.NewServer(http.FileServer(http.Dir(finalOutput)))
	defer server.Close()

	done := make(chan struct{})
	result := make(chan cleanPublishFetchResult, 1)
	go fetchHTMLAndPaletteUntilDone(t.Context(), server, done, result)

	for generation := 1; generation <= 50; generation++ {
		stagedOutput := filepath.Join(root, fmt.Sprintf("stage-%d", generation))
		writePublishGeneration(t, stagedOutput, generation)
		if err := retainPreviousGenerationBrowserAssets(finalOutput, stagedOutput); err != nil {
			t.Fatalf("retain generation %d assets: %v", generation, err)
		}
		if err := publishStagedOutput(stagedOutput, finalOutput); err != nil {
			t.Fatalf("publish generation %d: %v", generation, err)
		}
	}

	close(done)
	fetchResult := <-result
	if fetchResult.err != nil {
		t.Fatal(fetchResult.err)
	}
	if fetchResult.checks == 0 {
		t.Fatal("fetch loop did not complete an HTML + palette request")
	}
}

type cleanPublishFetchResult struct {
	checks int
	err    error
}

func fetchHTMLAndPaletteUntilDone(ctx context.Context, server *httptest.Server, done <-chan struct{}, result chan<- cleanPublishFetchResult) {
	checks := 0
	for {
		select {
		case <-done:
			result <- cleanPublishFetchResult{checks: checks}
			return
		default:
		}

		paletteRef, err := fetchPaletteReference(ctx, server.Client(), server.URL+"/")
		if err != nil {
			result <- cleanPublishFetchResult{checks: checks, err: err}
			return
		}
		if err := fetchSuccessful(ctx, server.Client(), server.URL+paletteRef); err != nil {
			result <- cleanPublishFetchResult{
				checks: checks,
				err:    fmt.Errorf("palette %s did not match successful HTML generation: %w", paletteRef, err),
			}
			return
		}
		checks++
	}
}

func fetchPaletteReference(ctx context.Context, client *http.Client, pageURL string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("create HTML request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("fetch HTML: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch HTML: status %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("read HTML: %w", err)
	}
	match := cleanPublishHTMLAssetPattern.FindSubmatch(body)
	if len(match) != 2 {
		return "", fmt.Errorf("HTML did not contain a stylesheet reference: %q", body)
	}
	return string(match[1]), nil
}

func fetchSuccessful(ctx context.Context, client *http.Client, assetURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("create asset request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", response.StatusCode)
	}
	return nil
}

func writePublishGeneration(t *testing.T, output string, generation int) {
	t.Helper()
	palette := fmt.Sprintf("palette.%08x.css", generation)
	mustWriteTestFile(t, filepath.Join(output, "index.html"), fmt.Sprintf(`<!doctype html><link rel="stylesheet" href="/css/%s">`, palette))
	mustWriteTestFile(t, filepath.Join(output, "css", palette), fmt.Sprintf(":root{--generation:%d}", generation))
}

func requireAtomicDirectoryExchange(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	left := filepath.Join(root, "left")
	right := filepath.Join(root, "right")
	if err := os.Mkdir(left, 0o755); err != nil {
		t.Fatalf("create exchange probe: %v", err)
	}
	if err := os.Mkdir(right, 0o755); err != nil {
		t.Fatalf("create exchange probe: %v", err)
	}
	if err := exchangeOutputDirectories(left, right); errors.Is(err, errAtomicExchangeUnsupported) {
		t.Skip("filesystem does not support atomic directory exchange")
	} else if err != nil {
		t.Fatalf("probe atomic directory exchange: %v", err)
	}
}
