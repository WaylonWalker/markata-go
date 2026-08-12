package renderingrecipe

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCanonicalBundle_AttachmentProbes exercises the same HTTP attachment path
// a browser uses, without changing the site generator or migrating recipes.
func TestCanonicalBundle_AttachmentProbes(t *testing.T) {
	theme, err := LoadCanonicalTheme()
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := Compile(theme)
	if err != nil {
		t.Fatal(err)
	}
	assets := bundle.Assets
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := assets[r.URL.Path[1:]]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write(data)
	}))
	defer server.Close()

	for _, asset := range bundle.Manifest.Assets {
		response, err := http.Get(server.URL + "/" + asset.Path)
		if err != nil {
			t.Fatalf("probe %s: %v", asset.Path, err)
		}
		data, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read probe %s: read=%v close=%v", asset.Path, readErr, closeErr)
		}
		if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "image/svg+xml" {
			t.Fatalf("probe %s: status=%d content-type=%q", asset.Path, response.StatusCode, response.Header.Get("Content-Type"))
		}
		digest := sha256.Sum256(data)
		if got := hex.EncodeToString(digest[:]); got != asset.SHA256 {
			t.Fatalf("probe %s: hash=%s manifest=%s", asset.Path, got, asset.SHA256)
		}
	}

	for _, pass := range bundle.Manifest.Passes {
		if _, ok := assets[pass.Asset]; !ok {
			t.Fatalf("pass %s references unattached asset %s", pass.ID, pass.Asset)
		}
	}
	for _, asset := range bundle.Manifest.Assets {
		if asset.Path == "assets/motif-block-w-v1.svg" && (asset.ViewBox != "0 0 28480 10060" || asset.Width != 28480 || asset.Height != 10060) {
			t.Fatalf("motif geometry metadata drifted: %+v", asset)
		}
	}
}
