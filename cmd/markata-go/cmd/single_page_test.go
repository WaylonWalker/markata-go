package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCreateSinglePageManager_BuildsOnlyRootIndex(t *testing.T) {
	site := t.TempDir()
	t.Chdir(site)
	if err := os.MkdirAll(filepath.Join(site, "pages"), 0o755); err != nil {
		t.Fatal(err)
	}
	post := "---\ntitle: Solo Page\nslug: custom-slug\npublished: true\n---\n\n# Solo Page\n\nHello from one file.\n"
	if err := os.WriteFile(filepath.Join(site, "pages", "solo.md"), []byte(post), 0o600); err != nil {
		t.Fatal(err)
	}

	m, err := createSinglePageManager("", filepath.Join("pages", "solo.md"))
	if err != nil {
		t.Fatalf("createSinglePageManager() error = %v", err)
	}
	applyFastMode(m)
	if _, err := runBuild(m); err != nil {
		t.Fatalf("runBuild() error = %v", err)
	}

	outputDir := m.Config().OutputDir
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(site, outputDir)
	}
	htmlBytes, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatalf("expected root index.html: %v", err)
	}
	html := string(htmlBytes)
	if !strings.Contains(html, "Hello from one file.") {
		t.Errorf("root index.html does not contain the post body")
	}
	year := strconv.Itoa(time.Now().Year())
	if !strings.Contains(html, "&copy; "+year) {
		t.Errorf("footer copyright missing current year %s", year)
	}
	for _, unwanted := range []string{`href="/archive/"`, `href="/rss.xml"`, `id="pagefind-search"`} {
		if strings.Contains(html, unwanted) {
			t.Errorf("single-page output unexpectedly contains %s", unwanted)
		}
	}

	for _, rel := range []string{
		"custom-slug",
		"solo",
		"archive",
		"tags",
		"images",
		"sitemap.xml",
		"rss.xml",
		"atom.xml",
		"404.html",
		".well-known",
		"_pagefind",
	} {
		if _, err := os.Stat(filepath.Join(outputDir, rel)); err == nil {
			t.Errorf("single-page build unexpectedly wrote %s", rel)
		}
	}

	var htmlFiles []string
	walkErr := filepath.WalkDir(outputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".html") {
			rel, err := filepath.Rel(outputDir, path)
			if err != nil {
				return err
			}
			htmlFiles = append(htmlFiles, rel)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	if len(htmlFiles) != 1 || htmlFiles[0] != "index.html" {
		t.Errorf("html files = %v, want only index.html", htmlFiles)
	}
}
