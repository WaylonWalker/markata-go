package buildcache

// RebuildReason explains why a cached post cannot be reused.
type RebuildReason string

const (
	RebuildReasonNone            RebuildReason = ""
	RebuildReasonMissingEntry    RebuildReason = "missing_entry"
	RebuildReasonInputChanged    RebuildReason = "input_changed"
	RebuildReasonTemplateChanged RebuildReason = "template_changed"
)

// ReasonForRebuild reports the cache-local reason sourcePath needs rebuilding.
// It intentionally covers only state owned by buildcache. Higher-level callers
// can layer dependency, feed-membership, and rendered-context reasons on top.
// It shares one locked metadata lookup with boolean and batch checks. An empty
// cached input hash is a partial entry compared normally, not a corruption claim.
func (c *Cache) ReasonForRebuild(sourcePath, inputHash, template string) RebuildReason {
	if c == nil {
		return RebuildReasonMissingEntry
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.rebuildReasonLocked(sourcePath, inputHash, template)
}
