package renderingrecipe

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type motifSVGRoot struct {
	XMLName xml.Name `xml:"svg"`
	Paths   []struct {
		Index string `xml:"data-index,attr"`
		Fill  string `xml:"fill,attr"`
	} `xml:"path"`
}

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
	motifBytes := assets["assets/motif-block-w-v1.svg"]
	var motif motifSVGRoot
	if err := xml.Unmarshal(motifBytes, &motif); err != nil {
		t.Fatalf("decode compiled motif: %v", err)
	}
	if len(motif.Paths) != 160 {
		t.Fatalf("compiled motif mark count = %d, want 160", len(motif.Paths))
	}
	if motif.Paths[0].Fill != "#0f1219" {
		t.Fatalf("compiled motif paint = %q, want canonical 1%% owning-surface mix", motif.Paths[0].Fill)
	}

	for _, pass := range bundle.Manifest.Passes {
		if _, ok := assets[pass.Asset]; !ok {
			t.Fatalf("pass %s references unattached asset %s", pass.ID, pass.Asset)
		}
	}
	if got := []string{bundle.Manifest.Passes[0].ID, bundle.Manifest.Passes[1].ID}; got[0] != "surface-texture" || got[1] != "motif-over" {
		t.Fatalf("canonical over layer order = %v", got)
	}
	for _, asset := range bundle.Manifest.Assets {
		if asset.Path == "assets/motif-block-w-v1.svg" && (asset.ViewBox != "0 0 1328 507" || asset.Width != 1328 || asset.Height != 507) {
			t.Fatalf("motif geometry metadata drifted: %+v", asset)
		}
	}
}

func TestCanonicalBundle_AssetBytesAreAuthoritative(t *testing.T) {
	theme, err := LoadCanonicalTheme()
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := Compile(theme)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"assets/surface-screenprint-v1.svg": "e17ce82e807c5bae2aadcf0cbe183a92ee10f7652092372b2be627272bd83edd",
		"assets/heading-splatter-v1.svg":    "1bb8bf31924619059fa58ed4b28e47914a004cdde029d43f4fb564432d955dc9",
	}
	// The surface tile must remain transparent. The exact digest is asserted
	// below after the compiler output is frozen and copied to consumers.
	if bytes := bundle.Assets["assets/surface-screenprint-v1.svg"]; len(bytes) == 0 || strings.Contains(string(bytes), "<rect") {
		t.Fatal("surface texture must not contain an opaque background rectangle")
	}
	for path, expected := range want {
		if got := AssetHash(bundle.Assets[path]); got != expected {
			t.Fatalf("%s hash=%s want=%s", path, got, expected)
		}
	}
}

func TestCanonicalMotif_CalibrationPaintEndpoints(t *testing.T) {
	theme, err := LoadCanonicalTheme()
	if err != nil {
		t.Fatal(err)
	}
	for mix, want := range map[float64]string{0: "#0d1017", 1: "#bfbdb6"} {
		t.Run(fmt.Sprintf("mix-%v", mix), func(t *testing.T) {
			theme.Motif.ColorMix = mix
			bundle, err := Compile(theme)
			if err != nil {
				t.Fatal(err)
			}
			var motif motifSVGRoot
			if err := xml.Unmarshal(bundle.Assets["assets/motif-block-w-v1.svg"], &motif); err != nil {
				t.Fatal(err)
			}
			if len(motif.Paths) != 160 || motif.Paths[0].Fill != want {
				t.Fatalf("calibration mix %v: count=%d paint=%q, want 160/%q", mix, len(motif.Paths), motif.Paths[0].Fill, want)
			}
		})
	}
}
