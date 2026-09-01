package buildlab

// RequiredScenarios returns the named regression scenarios covered by the
// Build Lab smoke catalog.  Paths are intentionally conventional so a fixture
// can opt into the catalog without embedding shell commands in the test plan.
// Callers may replace the initial content with their fixture's exact text.
func RequiredScenarios() []Scenario {
	build := func(id string, operations ...Operation) Scenario {
		return Scenario{ID: id, Version: "1", Operations: operations}
	}
	return []Scenario{
		build("cold-build", Operation{Type: OpClearCache}, Operation{Type: OpClearOutput}, Operation{Type: OpBuild}),
		build("hot-no-op", Operation{Type: OpBuild}, Operation{Type: OpBuild}),
		build("source-content-change", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/foo.md", Old: "old", New: "new"}, Operation{Type: OpBuild}),
		build("frontmatter-title-change", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/foo.md", Old: "title: Old", New: "title: New"}, Operation{Type: OpBuild}),
		build("slug-change", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/foo.md", Old: "slug: old", New: "slug: new"}, Operation{Type: OpBuild}),
		build("tag-add", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/foo.md", Old: "tags: [one]", New: "tags: [one, two]"}, Operation{Type: OpBuild}),
		build("tag-remove", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/foo.md", Old: "tags: [one, two]", New: "tags: [one]"}, Operation{Type: OpBuild}),
		build("wikilink-add", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/foo.md", Old: "body", New: "body [[bar]]"}, Operation{Type: OpBuild}),
		build("wikilink-change", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/foo.md", Old: "[[bar]]", New: "[[baz]]"}, Operation{Type: OpBuild}),
		build("wikilink-remove", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/foo.md", Old: " [[bar]]", New: ""}, Operation{Type: OpBuild}),
		build("embed-add", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/foo.md", Old: "body", New: "body ![[bar]]"}, Operation{Type: OpBuild}),
		build("embed-change", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/foo.md", Old: "![[bar]]", New: "![[baz]]"}, Operation{Type: OpBuild}),
		build("embed-remove", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/foo.md", Old: " ![[bar]]", New: ""}, Operation{Type: OpBuild}),
		build("embed-target-change", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/bar.md", Old: "title: Bar", New: "title: Changed"}, Operation{Type: OpBuild}),
		build("source-add", Operation{Type: OpBuild}, Operation{Type: OpWriteFile, Path: "content/new.md", Content: "---\ntitle: New\n---\nnew\n"}, Operation{Type: OpBuild}),
		build("source-delete", Operation{Type: OpBuild}, Operation{Type: OpDelete, Path: "content/foo.md"}, Operation{Type: OpBuild}),
		build("source-rename", Operation{Type: OpBuild}, Operation{Type: OpRename, Path: "content/foo.md", Dest: "content/renamed.md"}, Operation{Type: OpBuild}),
		build("template-change", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "templates/post.html", Old: "old", New: "new"}, Operation{Type: OpBuild}),
		build("config-change", Operation{Type: OpBuild}, Operation{Type: OpSetConfig, Path: "markata-go.toml", Key: "title", Value: "Changed"}, Operation{Type: OpBuild}),
		build("static-asset-change", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "static/site.css", Old: "old", New: "new"}, Operation{Type: OpBuild}),
		build("feed-membership-change", Operation{Type: OpBuild}, Operation{Type: OpReplaceExact, Path: "content/foo.md", Old: "tags: [one]", New: "tags: [two]"}, Operation{Type: OpBuild}),
		build("feed-filter-change", Operation{Type: OpBuild}, Operation{Type: OpSetConfig, Path: "markata-go.toml", Key: "filter", Value: "published == true"}, Operation{Type: OpBuild}),
		build("feed-sort-change", Operation{Type: OpBuild}, Operation{Type: OpSetConfig, Path: "markata-go.toml", Key: "sort", Value: "title"}, Operation{Type: OpBuild}),
		build("cache-cleared", Operation{Type: OpBuild}, Operation{Type: OpClearCache}, Operation{Type: OpBuild}),
	}
}
