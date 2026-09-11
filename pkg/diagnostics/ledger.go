package diagnostics

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Stable content reason codes. These codes are part of the build diagnostics
// contract; human-readable messages can change without changing the codes.
const (
	ReasonFrontmatterSuspiciousDelimiter = "frontmatter.suspicious_delimiter"
	ReasonFrontmatterLeadingWhitespace   = "frontmatter.leading_whitespace"
	ReasonFrontmatterMalformedClosing    = "frontmatter.malformed_closing_delimiter"
	ReasonFrontmatterMissingClosing      = "frontmatter.missing_closing_delimiter"
	ReasonFrontmatterParseError          = "frontmatter.parse_error"
	ReasonFrontmatterDuplicateKey        = "frontmatter.duplicate_key"
	ReasonFrontmatterInvalidType         = "frontmatter.invalid_type"
	ReasonFrontmatterInvalidDate         = "frontmatter.invalid_date"
	ReasonContentPublishedFalse          = "content.published_false"
	ReasonContentDraft                   = "content.draft"
	ReasonContentSkip                    = "content.skip"
	ReasonContentPrivate                 = "content.private"
	ReasonContentFiltered                = "content.filtered"
	ReasonContentDuplicateSlug           = "content.duplicate_slug"
	ReasonContentNoOutput                = "content.no_output"
	ReasonContentLoadError               = "content.load_error"
	ReasonContentRenderError             = "content.render_error"
	ReasonContentWriteError              = "content.write_error"
	ReasonFeedOffset                     = "feed.offset"
	ReasonFeedLimit                      = "feed.limit"
)

// ContentDispositionName identifies the final result for a source file.
const (
	DispositionEmitted      = "emitted"
	DispositionShadow       = "shadow"
	DispositionExcluded     = "excluded"
	DispositionNotCandidate = "not_candidate"
)

// ContentFeedDisposition records whether a source was selected for a feed.
type ContentFeedDisposition struct {
	Feed     string   `json:"feed"`
	Included bool     `json:"included"`
	Reasons  []string `json:"reasons,omitempty"`
}

// ContentDisposition records the state of one discovered source file.
type ContentDisposition struct {
	Path               string                   `json:"path"`
	Candidate          bool                     `json:"candidate"`
	Loaded             bool                     `json:"loaded"`
	FrontmatterPresent bool                     `json:"frontmatter_present"`
	FrontmatterValid   bool                     `json:"frontmatter_valid"`
	PostCreated        bool                     `json:"post_created"`
	Eligible           bool                     `json:"eligible"`
	Rendered           bool                     `json:"rendered"`
	Emitted            bool                     `json:"emitted"`
	Excluded           bool                     `json:"excluded"`
	Disposition        string                   `json:"disposition"`
	Reasons            []string                 `json:"reasons,omitempty"`
	Diagnostics        []Issue                  `json:"diagnostics,omitempty"`
	Feeds              []ContentFeedDisposition `json:"feeds,omitempty"`
}

// ContentSummary contains counts derived from a content ledger snapshot.
type ContentSummary struct {
	Discovered       int `json:"discovered"`
	Candidates       int `json:"candidates"`
	Loaded           int `json:"loaded"`
	FrontmatterValid int `json:"frontmatter_valid"`
	Posts            int `json:"posts"`
	Eligible         int `json:"eligible"`
	Rendered         int `json:"rendered"`
	Emitted          int `json:"emitted"`
	Excluded         int `json:"excluded"`
	Warnings         int `json:"warnings"`
	Errors           int `json:"errors"`
}

// ContentLedgerSnapshot is the immutable, deterministic view of a build's
// content state. Entries are sorted by source path.
type ContentLedgerSnapshot struct {
	Summary ContentSummary       `json:"summary"`
	Entries []ContentDisposition `json:"entries"`
}

type contentLedgerEntry struct {
	ContentDisposition
	feeds     map[string]*ContentFeedDisposition
	issueKeys map[string]struct{}
}

// ContentLedger is the canonical, concurrency-safe content state ledger for a
// build. Plugins record observations through this type; consumers use Snapshot.
type ContentLedger struct {
	mu      sync.RWMutex
	entries map[string]*contentLedgerEntry
}

// NewContentLedger creates an empty content ledger.
func NewContentLedger() *ContentLedger {
	return &ContentLedger{entries: make(map[string]*contentLedgerEntry)}
}

// Discover replaces the discovered file set for the current build.
func (l *ContentLedger) Discover(paths []string) {
	if l == nil {
		return
	}

	entries := make(map[string]*contentLedgerEntry, len(paths))
	for _, path := range paths {
		path = normalizeContentPath(path)
		if path == "" {
			continue
		}
		if _, exists := entries[path]; exists {
			continue
		}
		entries[path] = newContentLedgerEntry(path)
	}

	l.mu.Lock()
	l.entries = entries
	l.mu.Unlock()
}

// AddDiscovered adds one file to the discovered file set.
func (l *ContentLedger) AddDiscovered(path string) {
	if l == nil {
		return
	}
	path = normalizeContentPath(path)
	if path == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.entries == nil {
		l.entries = make(map[string]*contentLedgerEntry)
	}
	if _, exists := l.entries[path]; !exists {
		l.entries[path] = newContentLedgerEntry(path)
	}
}

// Reset clears all observations while retaining the ledger instance.
func (l *ContentLedger) Reset() {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.entries = make(map[string]*contentLedgerEntry)
	l.mu.Unlock()
}

// MarkLoaded records that source bytes were read successfully.
func (l *ContentLedger) MarkLoaded(path string) {
	l.withCandidate(path, func(entry *contentLedgerEntry) {
		entry.Loaded = true
	})
}

// MarkFrontmatter records frontmatter presence and parse validity.
func (l *ContentLedger) MarkFrontmatter(path string, present, valid bool) {
	l.withCandidate(path, func(entry *contentLedgerEntry) {
		entry.FrontmatterPresent = present
		entry.FrontmatterValid = valid
	})
}

// MarkPost records that a source file became a Post and whether it is eligible
// for public published content.
func (l *ContentLedger) MarkPost(path string, eligible bool) {
	l.withCandidate(path, func(entry *contentLedgerEntry) {
		entry.PostCreated = true
		entry.Eligible = eligible
	})
}

// MarkRendered records that Markdown HTML was produced or restored.
func (l *ContentLedger) MarkRendered(path string) {
	l.withCandidate(path, func(entry *contentLedgerEntry) {
		entry.Rendered = true
	})
}

// MarkEmitted records that at least one output for the source exists or was
// produced by the current write stage.
func (l *ContentLedger) MarkEmitted(path string) {
	l.withCandidate(path, func(entry *contentLedgerEntry) {
		entry.Emitted = true
	})
}

// AddReason records a stable reason code for a source file.
func (l *ContentLedger) AddReason(path, reason string) {
	if l == nil || reason == "" {
		return
	}
	l.withCandidate(path, func(entry *contentLedgerEntry) {
		for _, existing := range entry.Reasons {
			if existing == reason {
				return
			}
		}
		entry.Reasons = append(entry.Reasons, reason)
	})
}

// AddIssue records a source diagnostic and maps its code to a ledger reason.
// Duplicate diagnostics with the same location, code, and message are ignored.
func (l *ContentLedger) AddIssue(issue Issue) {
	if l == nil {
		return
	}
	issue.File = normalizeContentPath(issue.File)
	if issue.File == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	entry := l.entryLocked(issue.File)
	if !entry.Candidate {
		return
	}
	if entry.issueKeys == nil {
		entry.issueKeys = make(map[string]struct{})
	}
	key := issueIdentity(issue)
	if _, exists := entry.issueKeys[key]; exists {
		return
	}
	entry.issueKeys[key] = struct{}{}
	entry.Diagnostics = append(entry.Diagnostics, issue)
	if reason := ReasonForDiagnosticCode(issue.Code); reason != "" {
		addReasonLocked(entry, reason)
	}
}

// RecordError records a non-secret, source-scoped error diagnostic.
func (l *ContentLedger) RecordError(path, code, message string) {
	if code == "" {
		code = ReasonContentLoadError
	}
	l.AddIssue(Issue{
		File:     path,
		Code:     code,
		Severity: SeverityError,
		Message:  message,
	})
}

// HasIssueCode reports whether a source already has a diagnostic code.
func (l *ContentLedger) HasIssueCode(path, code string) bool {
	if l == nil || code == "" {
		return false
	}
	path = normalizeContentPath(path)
	l.mu.RLock()
	defer l.mu.RUnlock()
	entry, ok := l.entries[path]
	if !ok {
		return false
	}
	for _, issue := range entry.Diagnostics {
		if issue.Code == code {
			return true
		}
	}
	return false
}

// RecordFeed records per-feed selection for a source file.
func (l *ContentLedger) RecordFeed(path, feed string, included bool, reasons ...string) {
	if l == nil {
		return
	}
	path = normalizeContentPath(path)
	if path == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	entry := l.entryLocked(path)
	if !entry.Candidate {
		return
	}
	if entry.feeds == nil {
		entry.feeds = make(map[string]*ContentFeedDisposition)
	}
	feedDisposition, ok := entry.feeds[feed]
	if !ok {
		feedDisposition = &ContentFeedDisposition{Feed: feed}
		entry.feeds[feed] = feedDisposition
	}
	feedDisposition.Included = included
	for _, reason := range reasons {
		addFeedReason(feedDisposition, reason)
		if reason != "" && !included {
			addReasonLocked(entry, reason)
		}
	}
}

// Snapshot returns a stable copy of all content observations and derived
// counts. It is safe to call while worker pools are still finishing.
func (l *ContentLedger) Snapshot() ContentLedgerSnapshot {
	if l == nil {
		return ContentLedgerSnapshot{Entries: []ContentDisposition{}}
	}

	l.mu.RLock()
	entries := make([]*contentLedgerEntry, 0, len(l.entries))
	for _, entry := range l.entries {
		entries = append(entries, cloneContentLedgerEntry(entry))
	}
	l.mu.RUnlock()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	snapshot := ContentLedgerSnapshot{
		Entries: make([]ContentDisposition, 0, len(entries)),
	}
	for _, entry := range entries {
		disposition := entry.ContentDisposition
		disposition.Reasons = sortedUnique(append([]string{}, disposition.Reasons...))
		disposition.Diagnostics = sortedIssues(disposition.Diagnostics)
		disposition.Feeds = sortedFeeds(entry.feeds)

		snapshot.Summary.Discovered++
		if !disposition.Candidate {
			disposition.Disposition = DispositionNotCandidate
			snapshot.Entries = append(snapshot.Entries, disposition)
			continue
		}

		updateContentSummary(&snapshot.Summary, disposition)
		finalizeContentDisposition(&snapshot.Summary, &disposition)

		snapshot.Entries = append(snapshot.Entries, disposition)
	}

	return snapshot
}

func updateContentSummary(summary *ContentSummary, disposition ContentDisposition) {
	summary.Candidates++
	if disposition.Loaded {
		summary.Loaded++
	}
	if disposition.FrontmatterValid {
		summary.FrontmatterValid++
	}
	if disposition.PostCreated {
		summary.Posts++
	}
	if disposition.Eligible {
		summary.Eligible++
	}
	if disposition.Rendered {
		summary.Rendered++
	}
	if disposition.Emitted {
		summary.Emitted++
	}
	for _, issue := range disposition.Diagnostics {
		switch issue.Severity {
		case SeverityInfo:
			// Informational diagnostics are not warnings or errors.
		case SeverityWarning:
			summary.Warnings++
		case SeverityError:
			summary.Errors++
		}
	}
}

func finalizeContentDisposition(summary *ContentSummary, disposition *ContentDisposition) {
	if !disposition.Emitted && !hasTerminalExclusion(disposition.Reasons) {
		disposition.Reasons = append(disposition.Reasons, ReasonContentNoOutput)
		disposition.Reasons = sortedUnique(disposition.Reasons)
	}
	disposition.Excluded = !disposition.Eligible || !disposition.Emitted
	if disposition.Emitted {
		if disposition.Eligible {
			disposition.Disposition = DispositionEmitted
		} else {
			disposition.Disposition = DispositionShadow
		}
	} else {
		disposition.Disposition = DispositionExcluded
	}
	if disposition.Excluded {
		summary.Excluded++
	}
}

// ReasonForDiagnosticCode maps shared diagnostic codes to content reason codes.
func ReasonForDiagnosticCode(code string) string {
	switch code {
	case "duplicate-key":
		return ReasonFrontmatterDuplicateKey
	case "invalid-date":
		return ReasonFrontmatterInvalidDate
	case ReasonFrontmatterSuspiciousDelimiter,
		ReasonFrontmatterLeadingWhitespace,
		ReasonFrontmatterMalformedClosing,
		ReasonFrontmatterMissingClosing,
		ReasonFrontmatterParseError,
		ReasonFrontmatterInvalidType,
		ReasonContentPublishedFalse,
		ReasonContentDraft,
		ReasonContentSkip,
		ReasonContentPrivate,
		ReasonContentFiltered,
		ReasonContentDuplicateSlug,
		ReasonContentNoOutput,
		ReasonContentLoadError,
		ReasonContentRenderError,
		ReasonContentWriteError,
		ReasonFeedOffset,
		ReasonFeedLimit:
		return code
	default:
		if code == "" {
			return ""
		}
		return "diagnostic." + strings.ReplaceAll(code, "-", "_")
	}
}

// IsContentCandidate reports whether a path uses a source content extension.
func IsContentCandidate(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".mdown", ".mkd", ".mdx", ".rst", ".asciidoc", ".adoc", ".txt", ".text", ".html", ".htm":
		return true
	default:
		return false
	}
}

func newContentLedgerEntry(path string) *contentLedgerEntry {
	return &contentLedgerEntry{
		ContentDisposition: ContentDisposition{
			Path:      path,
			Candidate: IsContentCandidate(path),
		},
		feeds:     make(map[string]*ContentFeedDisposition),
		issueKeys: make(map[string]struct{}),
	}
}

func (l *ContentLedger) withCandidate(path string, fn func(*contentLedgerEntry)) {
	if l == nil || fn == nil {
		return
	}
	path = normalizeContentPath(path)
	if path == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := l.entryLocked(path)
	if entry.Candidate {
		fn(entry)
	}
}

func (l *ContentLedger) entryLocked(path string) *contentLedgerEntry {
	if l.entries == nil {
		l.entries = make(map[string]*contentLedgerEntry)
	}
	entry, ok := l.entries[path]
	if !ok {
		entry = newContentLedgerEntry(path)
		l.entries[path] = entry
	}
	return entry
}

func cloneContentLedgerEntry(entry *contentLedgerEntry) *contentLedgerEntry {
	clone := &contentLedgerEntry{
		ContentDisposition: entry.ContentDisposition,
		feeds:              make(map[string]*ContentFeedDisposition, len(entry.feeds)),
		issueKeys:          make(map[string]struct{}, len(entry.issueKeys)),
	}
	clone.Reasons = append([]string{}, entry.Reasons...)
	clone.Diagnostics = append([]Issue{}, entry.Diagnostics...)
	for name, feed := range entry.feeds {
		feedClone := *feed
		feedClone.Reasons = append([]string{}, feed.Reasons...)
		clone.feeds[name] = &feedClone
	}
	for key := range entry.issueKeys {
		clone.issueKeys[key] = struct{}{}
	}
	return clone
}

func addReasonLocked(entry *contentLedgerEntry, reason string) {
	if reason == "" {
		return
	}
	for _, existing := range entry.Reasons {
		if existing == reason {
			return
		}
	}
	entry.Reasons = append(entry.Reasons, reason)
}

func addFeedReason(feed *ContentFeedDisposition, reason string) {
	if feed == nil || reason == "" {
		return
	}
	for _, existing := range feed.Reasons {
		if existing == reason {
			return
		}
	}
	feed.Reasons = append(feed.Reasons, reason)
}

func issueIdentity(issue Issue) string {
	return fmt.Sprintf("%s:%d:%d:%d:%d:%s:%s", issue.Code, issue.Range.StartLine, issue.Range.StartCol, issue.Range.EndLine, issue.Range.EndCol, issue.Severity.String(), issue.Message)
}

func normalizeContentPath(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." {
		return ""
	}
	if filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, "../") {
		return redactedExternalPath(path)
	}
	return path
}

func redactedExternalPath(path string) string {
	base := filepath.Base(path)
	if base == "." || base == string(filepath.Separator) || base == "" {
		base = "source"
	}
	digest := sha256.Sum256([]byte(path))
	return filepath.ToSlash(filepath.Join("__outside_content_root__", fmt.Sprintf("%s-%x", base, digest[:4])))
}

func sortedUnique(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	sort.Strings(values)
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func sortedIssues(issues []Issue) []Issue {
	if len(issues) == 0 {
		return nil
	}
	sort.Slice(issues, func(i, j int) bool {
		left, right := issues[i], issues[j]
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Range.StartLine != right.Range.StartLine {
			return left.Range.StartLine < right.Range.StartLine
		}
		if left.Range.StartCol != right.Range.StartCol {
			return left.Range.StartCol < right.Range.StartCol
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		return left.Message < right.Message
	})
	return issues
}

func sortedFeeds(feeds map[string]*ContentFeedDisposition) []ContentFeedDisposition {
	if len(feeds) == 0 {
		return nil
	}
	names := make([]string, 0, len(feeds))
	for name := range feeds {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]ContentFeedDisposition, 0, len(names))
	for _, name := range names {
		feed := *feeds[name]
		feed.Reasons = sortedUnique(append([]string{}, feed.Reasons...))
		result = append(result, feed)
	}
	return result
}

func hasTerminalExclusion(reasons []string) bool {
	for _, reason := range reasons {
		switch reason {
		case ReasonContentDraft,
			ReasonContentSkip,
			ReasonFrontmatterParseError,
			ReasonFrontmatterMissingClosing,
			ReasonFrontmatterMalformedClosing,
			ReasonContentLoadError,
			ReasonContentRenderError,
			ReasonContentWriteError,
			ReasonContentDuplicateSlug:
			return true
		}
	}
	return false
}
