package cmd

import "testing"

func TestIncrementalPathsRequireFullRebuild(t *testing.T) {
	if incrementalPathsRequireFullRebuild([]string{"pages/post.md"}, nil) {
		t.Fatal("markdown content change forced a full rebuild")
	}
	for _, path := range []string{"markata-go.toml", "templates/base.html", "static/site.css"} {
		if !incrementalPathsRequireFullRebuild([]string{path}, nil) {
			t.Fatalf("global input %q did not force a full rebuild", path)
		}
	}
	if !incrementalPathsRequireFullRebuild(nil, []string{"pages/removed.md"}) {
		t.Fatal("removed content did not force a full rebuild")
	}
}
