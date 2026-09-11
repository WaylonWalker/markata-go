package diagnostics

import (
	"encoding/json"
	"fmt"
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
	Summary       ContentSummary       `json:"summary"`
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
		BuiltAt: builtAt,
		Summary: snapshot.Summary,
		Entries: cloneArtifactEntries(snapshot.Entries),
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
		result[index] = entry
		result[index].Path = normalizeContentPath(entry.Path)
		result[index].Reasons = append([]string(nil), entry.Reasons...)
		result[index].Diagnostics = append([]Issue(nil), entry.Diagnostics...)
		result[index].Feeds = append([]ContentFeedDisposition(nil), entry.Feeds...)
		result[index].Reasons = sortedUnique(result[index].Reasons)
		for diagnosticIndex := range result[index].Diagnostics {
			result[index].Diagnostics[diagnosticIndex] = sanitizeArtifactIssue(result[index].Diagnostics[diagnosticIndex])
		}
		result[index].Diagnostics = sortedIssues(result[index].Diagnostics)
		for feedIndex := range result[index].Feeds {
			result[index].Feeds[feedIndex].Reasons = append([]string(nil), result[index].Feeds[feedIndex].Reasons...)
			result[index].Feeds[feedIndex].Reasons = sortedUnique(result[index].Feeds[feedIndex].Reasons)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
	for index := range result {
		sort.SliceStable(result[index].Feeds, func(i, j int) bool {
			return result[index].Feeds[i].Feed < result[index].Feeds[j].Feed
		})
	}
	return result
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
