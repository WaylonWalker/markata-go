package buildcache

import "testing"

func TestGetPostSemanticHashBaseline_Availability(t *testing.T) {
	for _, tc := range []struct {
		name      string
		entry     *PostCache
		available bool
	}{
		{name: "absent new entry", available: true},
		{name: "empty new entry", entry: &PostCache{}, available: true},
		{name: "partial new entry", entry: &PostCache{FeedItemHash: "feed"}, available: true},
		{name: "complete established entry", entry: &PostCache{InputHash: "input", FeedItemHash: "feed", TagIndexHash: "tag", GardenHash: "garden"}, available: true},
		{name: "cold reset established entry", entry: &PostCache{InputHash: "input", FullHTMLPath: "full-page"}, available: false},
		{name: "missing feed hash", entry: &PostCache{InputHash: "input", TagIndexHash: "tag", GardenHash: "garden"}, available: false},
		{name: "missing tag hash", entry: &PostCache{InputHash: "input", FeedItemHash: "feed", GardenHash: "garden"}, available: false},
		{name: "missing garden hash", entry: &PostCache{InputHash: "input", FeedItemHash: "feed", TagIndexHash: "tag"}, available: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cache := New(t.TempDir())
			if tc.entry != nil {
				cache.Posts["post.md"] = tc.entry
			}
			feed, tag, garden, available := cache.GetPostSemanticHashBaseline("post.md")
			if available != tc.available {
				t.Fatalf("availability = %v, want %v", available, tc.available)
			}
			rawFeed, rawTag, rawGarden := cache.GetPostSemanticHashes("post.md")
			if feed != rawFeed || tag != rawTag || garden != rawGarden {
				t.Fatal("baseline availability changed the raw semantic getter")
			}
			if tc.entry != nil && (feed != tc.entry.FeedItemHash || tag != tc.entry.TagIndexHash || garden != tc.entry.GardenHash) {
				t.Fatal("baseline getter changed retained hashes")
			}
		})
	}
}
