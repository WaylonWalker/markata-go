package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveBrowserAssetPath(t *testing.T) {
	tests := []struct {
		name   string
		ref    string
		base   string
		want   string
		wantOK bool
	}{
		{name: "root css", ref: "/css/palette.deadbeef.css", base: "posts/example/index.html", want: filepath.FromSlash("css/palette.deadbeef.css"), wantOK: true},
		{name: "relative font", ref: "../fonts/site.woff2?v=1", base: "css/theme.css", want: filepath.FromSlash("fonts/site.woff2"), wantOK: true},
		{name: "relative js", ref: "../../assets/app.12345678.js", base: "posts/example/index.html", want: filepath.FromSlash("assets/app.12345678.js"), wantOK: true},
		{name: "external", ref: "https://example.com/site.css", base: "index.html", wantOK: false},
		{name: "protocol relative", ref: "//example.com/site.css", base: "index.html", wantOK: false},
		{name: "image ignored", ref: "/images/hero.jpg", base: "index.html", wantOK: false},
		{name: "traversal", ref: "../../../outside.css", base: "post/index.html", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveBrowserAssetPath(tt.ref, tt.base)
			if ok != tt.wantOK {
				t.Fatalf("resolveBrowserAssetPath(%q) ok = %v, want %v", tt.ref, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("resolveBrowserAssetPath(%q) = %q, want %q", tt.ref, got, tt.want)
			}
		})
	}
}

func TestRetainPreviousGenerationBrowserAssets(t *testing.T) {
	root := t.TempDir()
	previous := filepath.Join(root, "output")
	staged := filepath.Join(root, "staged")

	mustWriteTestFile(t, filepath.Join(previous, "post", "index.html"), `<!doctype html>
<link rel="stylesheet" href="/css/palette.deadbeef.css">
<script src="/js/site.12345678.js"></script>`)
	mustWriteTestFile(t, filepath.Join(previous, "css", "palette.deadbeef.css"), `@import "fonts.css";
@font-face { src: url("../fonts/site.woff2") format("woff2"); }
:root{--color-primary:#a54d68}`)
	mustWriteTestFile(t, filepath.Join(previous, "css", "fonts.css"), `@font-face { src: url("../fonts/secondary.woff2") format("woff2"); }`)
	mustWriteTestFile(t, filepath.Join(previous, "js", "site.12345678.js"), "console.log('old')")
	mustWriteTestFile(t, filepath.Join(previous, "fonts", "site.woff2"), "old font")
	mustWriteTestFile(t, filepath.Join(previous, "fonts", "secondary.woff2"), "secondary font")

	recentPath := filepath.Join(previous, "css", "palette.recent.css")
	mustWriteTestFile(t, recentPath, "grace-retained by the immediately previous generation")
	recentTime := time.Now().Add(-cleanPublishAssetGrace / 2)
	mustSetTestModTime(t, recentPath, recentTime)

	ancientPath := filepath.Join(previous, "css", "palette.ancient.css")
	mustWriteTestFile(t, ancientPath, "retained by an older generation")
	mustSetTestModTime(t, ancientPath, time.Now().Add(-cleanPublishAssetGrace-time.Minute))
	mustWriteTestFile(t, filepath.Join(previous, "images", "old.jpg"), "old image")

	mustWriteTestFile(t, filepath.Join(staged, "post", "index.html"), `<!doctype html><link rel="stylesheet" href="/css/palette.cafebabe.css">`)
	mustWriteTestFile(t, filepath.Join(staged, "css", "palette.cafebabe.css"), ":root{--color-primary:#123456}")

	if err := retainPreviousGenerationBrowserAssets(previous, staged); err != nil {
		t.Fatalf("retainPreviousGenerationBrowserAssets() error = %v", err)
	}

	for _, rel := range []string{
		"css/palette.deadbeef.css",
		"css/fonts.css",
		"css/palette.recent.css",
		"js/site.12345678.js",
		"fonts/site.woff2",
		"fonts/secondary.woff2",
	} {
		if _, err := os.Stat(filepath.Join(staged, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected retained asset %s: %v", rel, err)
		}
	}
	for _, rel := range []string{"css/palette.ancient.css", "images/old.jpg"} {
		if _, err := os.Stat(filepath.Join(staged, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("unexpected retained file %s", rel)
		}
	}

	recentInfo, err := os.Stat(filepath.Join(staged, "css", "palette.recent.css"))
	if err != nil {
		t.Fatalf("stat retained recent asset: %v", err)
	}
	if got := recentInfo.ModTime(); got.Before(recentTime.Add(-time.Second)) || got.After(recentTime.Add(time.Second)) {
		t.Fatalf("retained asset modtime = %v, want approximately %v", got, recentTime)
	}
}

func TestRetainPreviousGenerationStartsGraceWhenOldActiveAssetRetires(t *testing.T) {
	root := t.TempDir()
	previous := filepath.Join(root, "output")
	firstStage := filepath.Join(root, "stage-1")
	secondStage := filepath.Join(root, "stage-2")

	oldPalette := filepath.Join(previous, "css", "palette.deadbeef.css")
	mustWriteTestFile(t, filepath.Join(previous, "index.html"), `<!doctype html><link rel="stylesheet" href="/css/palette.deadbeef.css">`)
	mustWriteTestFile(t, oldPalette, "old active palette")
	mustSetTestModTime(t, oldPalette, time.Now().Add(-cleanPublishAssetGrace-time.Hour))

	mustWriteTestFile(t, filepath.Join(firstStage, "index.html"), `<!doctype html><link rel="stylesheet" href="/css/palette.cafebabe.css">`)
	mustWriteTestFile(t, filepath.Join(firstStage, "css", "palette.cafebabe.css"), "first replacement palette")

	retiredAfter := time.Now().Add(-time.Second)
	if err := retainPreviousGenerationBrowserAssets(previous, firstStage); err != nil {
		t.Fatalf("first retention error = %v", err)
	}
	firstInfo, err := os.Stat(filepath.Join(firstStage, "css", "palette.deadbeef.css"))
	if err != nil {
		t.Fatalf("stat newly retired palette: %v", err)
	}
	if firstInfo.ModTime().Before(retiredAfter) {
		t.Fatalf("newly retired palette kept original stale modtime %v, want retirement time after %v", firstInfo.ModTime(), retiredAfter)
	}

	mustWriteTestFile(t, filepath.Join(secondStage, "index.html"), `<!doctype html><link rel="stylesheet" href="/css/palette.facefeed.css">`)
	mustWriteTestFile(t, filepath.Join(secondStage, "css", "palette.facefeed.css"), "second replacement palette")
	if err := retainPreviousGenerationBrowserAssets(firstStage, secondStage); err != nil {
		t.Fatalf("second retention error = %v", err)
	}
	secondInfo, err := os.Stat(filepath.Join(secondStage, "css", "palette.deadbeef.css"))
	if err != nil {
		t.Fatalf("old active palette did not survive rapid second publish: %v", err)
	}
	if got := secondInfo.ModTime(); got.Before(firstInfo.ModTime().Add(-time.Second)) || got.After(firstInfo.ModTime().Add(time.Second)) {
		t.Fatalf("grace-retained palette modtime = %v, want retirement time approximately %v", got, firstInfo.ModTime())
	}
}

func TestPublishKeepsOldPaletteAvailableAfterGenerationSwitch(t *testing.T) {
	root := t.TempDir()
	finalOutput := filepath.Join(root, "output")
	stagedOutput := filepath.Join(root, "staged")

	oldHTML := `<!doctype html><link rel="stylesheet" href="/css/palette.deadbeef.css">`
	mustWriteTestFile(t, filepath.Join(finalOutput, "index.html"), oldHTML)
	mustWriteTestFile(t, filepath.Join(finalOutput, "css", "palette.deadbeef.css"), `@font-face { src: url("../fonts/old.woff2"); }`)
	mustWriteTestFile(t, filepath.Join(finalOutput, "fonts", "old.woff2"), "old font")

	mustWriteTestFile(t, filepath.Join(stagedOutput, "index.html"), `<!doctype html><link rel="stylesheet" href="/css/palette.cafebabe.css">`)
	mustWriteTestFile(t, filepath.Join(stagedOutput, "css", "palette.cafebabe.css"), "new palette")

	// Model the important race deterministically: the browser receives old HTML
	// immediately before publish, then asks for that HTML's stylesheet after the
	// completed generation has replaced the output directory.
	servedHTML, err := os.ReadFile(filepath.Join(finalOutput, "index.html"))
	if err != nil {
		t.Fatalf("read old HTML before publish: %v", err)
	}
	if string(servedHTML) != oldHTML {
		t.Fatalf("old HTML = %q, want %q", servedHTML, oldHTML)
	}

	if err := retainPreviousGenerationBrowserAssets(finalOutput, stagedOutput); err != nil {
		t.Fatalf("retainPreviousGenerationBrowserAssets() error = %v", err)
	}
	if err := publishStagedOutput(stagedOutput, finalOutput); err != nil {
		t.Fatalf("publishStagedOutput() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(finalOutput, "css", "palette.deadbeef.css")); err != nil {
		t.Fatalf("old HTML palette missing after publish: %v", err)
	}
	if _, err := os.Stat(filepath.Join(finalOutput, "fonts", "old.woff2")); err != nil {
		t.Fatalf("old font missing after publish: %v", err)
	}
	if _, err := os.Stat(filepath.Join(finalOutput, "css", "palette.cafebabe.css")); err != nil {
		t.Fatalf("new palette missing after publish: %v", err)
	}
}

func TestPublishStagedOutputReplacesGeneration(t *testing.T) {
	root := t.TempDir()
	finalOutput := filepath.Join(root, "output")
	stagedOutput := filepath.Join(root, "staged")

	mustWriteTestFile(t, filepath.Join(finalOutput, "index.html"), "old")
	mustWriteTestFile(t, filepath.Join(finalOutput, "stale.html"), "stale")
	mustWriteTestFile(t, filepath.Join(stagedOutput, "index.html"), "new")
	mustWriteTestFile(t, filepath.Join(stagedOutput, "css", "palette.cafebabe.css"), "new palette")

	if err := publishStagedOutput(stagedOutput, finalOutput); err != nil {
		t.Fatalf("publishStagedOutput() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(finalOutput, "index.html"))
	if err != nil {
		t.Fatalf("read published index: %v", err)
	}
	if string(content) != "new" {
		t.Fatalf("published index = %q, want %q", content, "new")
	}
	if _, err := os.Stat(filepath.Join(finalOutput, "stale.html")); !os.IsNotExist(err) {
		t.Fatalf("stale file survived generation replacement")
	}
	if _, err := os.Stat(stagedOutput); !os.IsNotExist(err) {
		t.Fatalf("staged directory still exists after publish")
	}
}

func TestPublishStagedOutputWithoutPreviousGeneration(t *testing.T) {
	root := t.TempDir()
	finalOutput := filepath.Join(root, "output")
	stagedOutput := filepath.Join(root, "staged")
	mustWriteTestFile(t, filepath.Join(stagedOutput, "index.html"), "first")

	if err := publishStagedOutput(stagedOutput, finalOutput); err != nil {
		t.Fatalf("publishStagedOutput() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(finalOutput, "index.html"))
	if err != nil {
		t.Fatalf("read published index: %v", err)
	}
	if string(content) != "first" {
		t.Fatalf("published index = %q, want %q", content, "first")
	}
}

func mustWriteTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustSetTestModTime(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("set modtime for %s: %v", path, err)
	}
}
