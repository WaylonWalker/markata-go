# Navigation Projection Experiment

This experimental specification covers issue #1528, not an enabled-by-default
feature. Normal static documents and links remain authoritative.

A projection MUST come from the canonical full render's existing
`#view-transition-page` region. It MUST use a stable `_nav.html` route URL and
content-derived revision over exact region bytes, metadata, and required assets.
No global build identifier or independent template rendering is permitted.

The experiment MUST retain one atomic region. Schema/shell compatibility is
separate from content freshness. A compatible newer route MUST remain usable
with an older open document. Unknown schema, shell, assets, or lifecycle MUST
fall back to ordinary navigation.

Publication MUST reuse the existing post/feed decisions. Unchanged projections
MUST not be written, even if the full publisher ran. Missing expected artifacts
MUST be repaired. Incremental experiments MUST count file writes and identify
legitimate dependents. Prototype coverage gaps MUST be reported.

Canonical region bytes MUST match exactly. Title, canonical URL, body state,
selected navigation, and required assets MUST come from canonical document state.
Persistent header/theme/adaptive state MUST not be reset by a successful swap.
A production implementation requires explicit cleanup/idempotent route setup,
asset initialization, focus, history, scroll, and adaptive route-entry semantics.

Measure broad artifact savings before implementing fragment splitting or a
router. Compare ordinary navigation, hx-boost/full document, HTMX projection,
and native projection with actual runtime byte costs. Weak measured savings
are a valid reason to stop. This experiment may conclude NOT JUSTIFIED or
NEEDS MORE EVIDENCE without making a production feature.
