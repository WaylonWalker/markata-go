package configdocs

import (
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
	if string(got) != string(want) {
		t.Fatal("pkg/config/settings_docs_gen.go is stale; run `go generate ./pkg/config`")
	}
}
