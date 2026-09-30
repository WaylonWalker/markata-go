package diagnostics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	// ArtifactSchemaURL identifies the versioned diagnostics wire format.
	ArtifactSchemaURL = "markata://schemas/content-diagnostics/v1"

	// ArtifactSchema is the stable diagnostics artifact identity.
	ArtifactSchema = "markata.content-diagnostics"

	// ArtifactSchemaVersion is the current diagnostics artifact generation.
	ArtifactSchemaVersion = 1

	// DefaultArtifactPath is relative to a build's output directory.
	DefaultArtifactPath = ".markata/diagnostics.json"

	// ArtifactExecutorLegacy identifies the normal lifecycle build path.
	ArtifactExecutorLegacy = "legacy"
	// ArtifactExecutorDAG identifies the feature-flagged task-graph build path.
	ArtifactExecutorDAG = "dag"
)

// Artifact describes the content state observed during one successful build.
// Its content fields are copied from a ContentLedgerSnapshot; they are not
// reconstructed from posts, feeds, or plugin-local counters.
type Artifact struct {
	SchemaURL     string               `json:"$schema"`
	Schema        string               `json:"schema"`
	SchemaVersion int                  `json:"schema_version"`
	Generator     ArtifactGenerator    `json:"generator"`
	Source        *ArtifactSource      `json:"source,omitempty"`
	BuiltAt       time.Time            `json:"built_at"`
	Executor      string               `json:"executor,omitempty"`
	Summary       ContentSummary       `json:"summary"`
	TemplateCache *TemplateCacheStats  `json:"template_cache,omitempty"`
	Entries       []ContentDisposition `json:"entries"`
}

// ArtifactGenerator identifies the markata-go binary that produced an
// artifact. Commit is omitted when the binary was not built with a reliable
// source revision.
type ArtifactGenerator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
}

// ArtifactSource identifies the source Git revision used for the build.
type ArtifactSource struct {
	Commit string `json:"commit"`
}

// ArtifactBuildInfo supplies build identity values for an artifact.
type ArtifactBuildInfo struct {
	MarkataVersion string
	MarkataCommit  string
	SourceCommit   string
	BuiltAt        time.Time
	Executor       string
}

// NewArtifact creates a versioned artifact from a deterministic ledger
// snapshot. A zero BuiltAt is replaced with the current UTC time.
func NewArtifact(snapshot ContentLedgerSnapshot, info ArtifactBuildInfo) Artifact {
	builtAt := info.BuiltAt
	if builtAt.IsZero() {
		builtAt = time.Now().UTC()
	} else {
		builtAt = builtAt.UTC()
	}

	version := strings.TrimSpace(info.MarkataVersion)
	if version == "" {
		version = "dev"
	}

	artifact := Artifact{
		SchemaURL:     ArtifactSchemaURL,
		Schema:        ArtifactSchema,
		SchemaVersion: ArtifactSchemaVersion,
		Generator: ArtifactGenerator{
			Name:    "markata-go",
			Version: version,
			Commit:  reliableCommit(info.MarkataCommit),
		},
		BuiltAt:       builtAt,
		Executor:      strings.TrimSpace(info.Executor),
		Summary:       snapshot.Summary,
		TemplateCache: cloneTemplateCacheStats(snapshot.TemplateCache),
		Entries:       cloneArtifactEntries(snapshot.Entries),
	}
	if commit := reliableCommit(info.SourceCommit); commit != "" {
		artifact.Source = &ArtifactSource{Commit: commit}
	}
	return artifact
}

// MarshalArtifact encodes a diagnostics artifact as indented, stable JSON.
func MarshalArtifact(snapshot ContentLedgerSnapshot, info ArtifactBuildInfo) ([]byte, error) {
	artifact := NewArtifact(snapshot, info)
	if err := validateArtifact(artifact); err != nil {
		return nil, err
	}
	return json.MarshalIndent(artifact, "", "  ")
}

// WriteArtifact streams the same indented JSON as MarshalArtifact, without a
// trailing newline or modifying snapshot. Temporary JSON buffers and sanitized
// copies are limited to one entry; the ordering index scales with entry count.
// Use a buffered writer when publishing to a file, and flush it before syncing.
func WriteArtifact(writer io.Writer, snapshot ContentLedgerSnapshot, info ArtifactBuildInfo) error {
	header := NewArtifact(ContentLedgerSnapshot{Summary: snapshot.Summary, TemplateCache: snapshot.TemplateCache}, info)
	if err := validateArtifact(header); err != nil {
		return err
	}
	data, err := json.MarshalIndent(header, "", "  ")
	if err != nil {
		return err
	}
	if len(snapshot.Entries) == 0 {
		return writeArtifactBytes(writer, data)
	}

	// Entries is the final field of Artifact. Verify the generated suffix rather
	// than silently corrupting output if that layout changes in the future.
	const emptyEntriesSuffix = "  \"entries\": []\n}"
	if !bytes.HasSuffix(data, []byte(emptyEntriesSuffix)) {
		return fmt.Errorf("diagnostics artifact header must end with empty entries")
	}
	if err := writeArtifactBytes(writer, data[:len(data)-len("]\n}")]); err != nil {
		return err
	}

	type entryIndex struct {
		path  string
		index int
	}
	order := make([]entryIndex, len(snapshot.Entries))
	for index, entry := range snapshot.Entries {
		order[index] = entryIndex{path: normalizeContentPath(entry.Path), index: index}
	}
	sort.SliceStable(order, func(i, j int) bool {
		return order[i].path < order[j].path
	})
	// Reuse both buffers across entries. Encoder supplies the same escaping as
	// Marshal, while Indent supplies exactly MarshalIndent's layout. Neither
	// buffer grows with the total number of entries.
	var compact, indented bytes.Buffer
	encoder := json.NewEncoder(&compact)
	for index, item := range order {
		entry := cloneArtifactEntry(snapshot.Entries[item.index])
		compact.Reset()
		if err := encoder.Encode(entry); err != nil {
			return err
		}
		indented.Reset()
		// Encode appends a newline; MarshalArtifact does not.
		encoded := bytes.TrimSuffix(compact.Bytes(), []byte("\n"))
		if err := json.Indent(&indented, encoded, "    ", "  "); err != nil {
			return err
		}
		separator := "\n    "
		if index > 0 {
			separator = ",\n    "
		}
		if err := writeArtifactBytes(writer, []byte(separator)); err != nil {
			return err
		}
		if err := writeArtifactBytes(writer, indented.Bytes()); err != nil {
			return err
		}
	}
	return writeArtifactBytes(writer, []byte("\n  ]\n}"))
}

func writeArtifactBytes(writer io.Writer, data []byte) error {
	written, err := writer.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

// ParseArtifact decodes and validates a diagnostics artifact.
func ParseArtifact(data []byte) (Artifact, error) {
	var artifact Artifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return Artifact{}, fmt.Errorf("decode diagnostics artifact JSON: %w", err)
	}
	if err := validateArtifact(artifact); err != nil {
		return Artifact{}, err
	}
	return artifact, nil
}

func validateArtifact(artifact Artifact) error {
	if artifact.SchemaURL != ArtifactSchemaURL {
		return fmt.Errorf("diagnostics artifact $schema must be %q", ArtifactSchemaURL)
	}
	if artifact.Schema != ArtifactSchema {
		return fmt.Errorf("diagnostics artifact schema must be %q", ArtifactSchema)
	}
	if artifact.SchemaVersion != ArtifactSchemaVersion {
		return fmt.Errorf("unsupported diagnostics artifact schema version %d", artifact.SchemaVersion)
	}
	if artifact.Generator.Name == "" {
		return fmt.Errorf("diagnostics artifact generator.name is required")
	}
	if artifact.Generator.Version == "" {
		return fmt.Errorf("diagnostics artifact generator.version is required")
	}
	if artifact.BuiltAt.IsZero() {
		return fmt.Errorf("diagnostics artifact built_at is required")
	}
	if artifact.Executor != "" && artifact.Executor != ArtifactExecutorLegacy && artifact.Executor != ArtifactExecutorDAG {
		return fmt.Errorf("diagnostics artifact executor must be %q or %q", ArtifactExecutorLegacy, ArtifactExecutorDAG)
	}
	if artifact.Source != nil && artifact.Source.Commit == "" {
		return fmt.Errorf("diagnostics artifact source.commit must not be empty")
	}
	if artifact.Entries == nil {
		return fmt.Errorf("diagnostics artifact entries is required")
	}
	return nil
}

func reliableCommit(commit string) string {
	commit = strings.TrimSpace(commit)
	switch commit {
	case "", "none", "unknown", "dev":
		return ""
	default:
		return commit
	}
}

func cloneArtifactEntries(entries []ContentDisposition) []ContentDisposition {
	if entries == nil {
		return []ContentDisposition{}
	}
	result := make([]ContentDisposition, len(entries))
	for index, entry := range entries {
		result[index] = cloneArtifactEntry(entry)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
	return result
}

// cloneArtifactEntry owns every mutable slice before normalizing or sorting it.
// Both artifact encoders use this helper so sanitation cannot drift.
func cloneArtifactEntry(entry ContentDisposition) ContentDisposition {
	entry.Path = normalizeContentPath(entry.Path)
	entry.Reasons = sortedUnique(append([]string(nil), entry.Reasons...))
	entry.Diagnostics = append([]Issue(nil), entry.Diagnostics...)
	for index := range entry.Diagnostics {
		entry.Diagnostics[index] = sanitizeArtifactIssue(entry.Diagnostics[index])
	}
	entry.Diagnostics = sortedIssues(entry.Diagnostics)
	entry.Feeds = append([]ContentFeedDisposition(nil), entry.Feeds...)
	for index := range entry.Feeds {
		entry.Feeds[index].Reasons = sortedUnique(append([]string(nil), entry.Feeds[index].Reasons...))
	}
	sort.SliceStable(entry.Feeds, func(i, j int) bool {
		return entry.Feeds[i].Feed < entry.Feeds[j].Feed
	})
	return entry
}

func sanitizeArtifactIssue(issue Issue) Issue {
	issue.File = normalizeContentPath(issue.File)
	// Built-in diagnostics use safe, concise messages. Do not copy arbitrary
	// producer-provided text into a public artifact because a third-party plugin
	// could otherwise publish raw content, configuration, or secrets.
	issue.Message = safeArtifactDiagnosticMessage(issue.Code)
	return issue
}

var safeArtifactDiagnosticMessages = map[string]string{
	ReasonFrontmatterSuspiciousDelimiter: "frontmatter opening delimiter is suspicious",
	ReasonFrontmatterLeadingWhitespace:   "frontmatter opening delimiter has leading whitespace",
	ReasonFrontmatterMalformedClosing:    "frontmatter closing delimiter is malformed",
	ReasonFrontmatterMissingClosing:      "frontmatter closing delimiter is missing",
	ReasonFrontmatterParseError:          "frontmatter could not be parsed",
	ReasonFrontmatterDuplicateKey:        "frontmatter contains a duplicate key",
	diagnosticCodeDuplicateKey:           "frontmatter contains a duplicate key",
	ReasonFrontmatterInvalidType:         "frontmatter field has an invalid type",
	ReasonFrontmatterInvalidDate:         "frontmatter date has an invalid format",
	diagnosticCodeInvalidDate:            "frontmatter date has an invalid format",
	ReasonContentPublishedFalse:          "content is not published",
	ReasonContentDraft:                   "content is a draft",
	ReasonContentSkip:                    "content is explicitly skipped",
	ReasonContentPrivate:                 "content is private",
	ReasonContentFiltered:                "content was filtered from the feed",
	ReasonContentDuplicateSlug:           "content has a duplicate slug",
	ReasonContentNoOutput:                "content has no output",
	ReasonContentLoadError:               "content could not be loaded",
	ReasonContentRenderError:             "content could not be rendered",
	ReasonContentWriteError:              "content output could not be written",
	ReasonFeedOffset:                     "content is outside the feed offset",
	ReasonFeedLimit:                      "content is outside the feed limit",
	diagnosticCodeMissingAltText:         "image link is missing alt text",
	diagnosticCodeProtocolLessURL:        "URL is missing a protocol",
	diagnosticCodeH1InContent:            "content contains an H1 heading",
	diagnosticCodeAdmonitionFence:        "fenced code follows an admonition without a blank line",
	diagnosticCodeBrokenWikilink:         "wikilink target was not found",
	diagnosticCodeUnknownMention:         "mention target was not found",
}

func safeArtifactDiagnosticMessage(code string) string {
	return safeArtifactDiagnosticMessages[code]
}
