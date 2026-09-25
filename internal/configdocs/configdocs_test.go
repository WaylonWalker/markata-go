package configdocs

import (
	"bytes"
	"os"
	"testing"
)

func TestSettingsDocsGenerated_UpToDate(t *testing.T) {
	docs, err := Extract("../../pkg/models")
	if err != nil {
		t.Fatal(err)
	}
	if docs["Config.Title"] == "" {
		t.Fatal("expected a doc comment for Config.Title")
	}
	want, err := Render("config", docs)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("../../pkg/config/settings_docs_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	// Windows checkouts may convert the generated file to CRLF.
	got = bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n"))
	if !bytes.Equal(got, bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))) {
		t.Fatal("pkg/config/settings_docs_gen.go is stale; run `go generate ./pkg/config`")
	}
}
