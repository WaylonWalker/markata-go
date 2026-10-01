package diagnostics

// TemplateCacheStats observes one completed template render invocation.
// It is bounded, contains no content identifiers, and is independent of profiling.
type TemplateCacheStats struct {
	Classified      int                      `json:"classified"`
	Skipped         int                      `json:"skipped"`
	Cacheable       int                      `json:"cacheable"`
	Restored        int                      `json:"restored"`
	RenderRequired  int                      `json:"render_required"`
	RenderSucceeded int                      `json:"render_succeeded"`
	RenderFailed    int                      `json:"render_failed"`
	ServeDeferred   int                      `json:"serve_deferred"`
	NavPreviewReset bool                     `json:"nav_preview_reset"`
	MissReasons     TemplateCacheMissReasons `json:"miss_reasons"`
}

// TemplateCacheMissReasons counts only the first failing gate in existing
// classification order. FullHTMLUnavailable is observed during restoration.
type TemplateCacheMissReasons struct {
	AffectedPath          int `json:"affected_path"`
	CacheUnavailable      int `json:"cache_unavailable"`
	InputHashMissing      int `json:"input_hash_missing"`
	EntryMissing          int `json:"entry_missing"`
	InputHashMismatch     int `json:"input_hash_mismatch"`
	TemplateMismatch      int `json:"template_mismatch"`
	DependencyChanged     int `json:"dependency_changed"`
	SlugChanged           int `json:"slug_changed"`
	FeedMembershipChanged int `json:"feed_membership_changed"`
	LocalPreviewChanged   int `json:"local_preview_changed"`
	FullHTMLUnavailable   int `json:"full_html_unavailable"`
}

func cloneTemplateCacheStats(stats *TemplateCacheStats) *TemplateCacheStats {
	if stats == nil {
		return nil
	}
	cloned := *stats
	return &cloned
}

// SetTemplateCache replaces the completed template observations, copying the
// value so callers retain no mutable reference to ledger state. Nil clears it.
func (l *ContentLedger) SetTemplateCache(stats *TemplateCacheStats) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.templateCache = cloneTemplateCacheStats(stats)
	l.mu.Unlock()
}
