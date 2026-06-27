# waylonwalker.com author loop benchmark (2026-06-26)

Baseline commit: `d7a64eb` with no working-tree changes
Candidate: current `markata/go-perf` working tree
Site: `../waylonwalker.com`
Command shape: `MARKATA_GO_BLOGROLL_ENABLED=false MARKATA_GO_ENCRYPTION_ENABLED=false markata-go build --fast -m fast.toml`

## Method

- Built a clean baseline binary from a detached worktree at `HEAD`.
- Built a candidate binary from the current working tree.
- For cold and warm timings, removed `output`, `cache`, `.markata`, `.markata-cache`, and `.markata.cache` before the first run.
- For changed-post timings, primed the cache, appended a temporary HTML comment to `pages/blog/ai.md`, ran one build, and restored the file.

## Results

| Scenario | Baseline | Candidate | Delta |
| --- | ---: | ---: | ---: |
| Cold | 2.18s | 1.74s | -20.2% |
| Warm 1 | 0.42s | 0.41s | -2.4% |
| Warm 2 | 0.42s | 0.40s | -4.8% |
| Changed post | 0.49s | 0.52s | +6.1% |

## Notes

- The biggest measured gain in this pass is lower cold-start overhead on the fast authoring config.
- Unchanged warm builds stayed roughly flat while remaining very fast.
- The changed-post delta is within noise for this small sample and should not be treated as a confirmed regression without repeated runs.
- Dominant hotspots during these runs were still `cdn_assets`, `link_avatars`, `render_markdown`, `tailwind`, and feed/listing writes after content edits.
- No live in-cluster timing was captured in this note.
