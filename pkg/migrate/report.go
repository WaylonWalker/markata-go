package migrate

import (
	"fmt"
	"strings"
)

// Report generates a human-readable migration report.
func (r *MigrationResult) Report() string {
	var sb strings.Builder

	// Header
	sb.WriteString(strings.Repeat("=", 80) + "\n")
	sb.WriteString("                        markata-go Migration Report\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")

	// File info
	if r.InputFile != "" {
		sb.WriteString(fmt.Sprintf("Configuration File: %s\n", r.InputFile))
	}
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", r.Timestamp.Format("2006-01-02 15:04:05")))

	// Summary
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString("SUMMARY\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n\n")

	status := "Ready to migrate"
	if r.HasErrors() {
		status = "Migration has issues"
	} else if r.HasWarnings() {
		status = "Ready to migrate (with warnings)"
	}
	sb.WriteString(fmt.Sprintf("  Status: %s\n\n", status))

	sb.WriteString(fmt.Sprintf("  Changes required:    %d\n", len(r.Changes)))
	sb.WriteString(fmt.Sprintf("  Filter migrations:   %d\n", len(r.FilterMigrations)))
	sb.WriteString(fmt.Sprintf("  Warnings:            %d\n", len(r.Warnings)))
	sb.WriteString(fmt.Sprintf("  Errors:              %d\n", len(r.Errors)))

	if len(r.TemplateIssues) > 0 {
		sb.WriteString(fmt.Sprintf("  Template issues:     %d\n", len(r.TemplateIssues)))
	}

	sb.WriteString("\n")

	// Configuration Changes
	if len(r.Changes) > 0 {
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		sb.WriteString("CONFIGURATION CHANGES\n")
		sb.WriteString(strings.Repeat("-", 80) + "\n\n")

		for _, change := range r.Changes {
			icon := "[MIGRATE]"
			sb.WriteString(fmt.Sprintf("  %s %s\n", icon, change.Description))
		}
		sb.WriteString("\n")
	}

	// Filter Migrations
	if len(r.FilterMigrations) > 0 {
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		sb.WriteString("FILTER MIGRATIONS\n")
		sb.WriteString(strings.Repeat("-", 80) + "\n\n")

		for _, fm := range r.FilterMigrations {
			feedName := fm.Feed
			if feedName == "" {
				feedName = "(unnamed)"
			}
			sb.WriteString(fmt.Sprintf("  Feed: %s\n", feedName))

			if len(fm.Changes) == 0 {
				sb.WriteString(fmt.Sprintf("    [OK] %s (no changes needed)\n", fm.Original))
			} else {
				sb.WriteString(fmt.Sprintf("    [MIGRATE] %s\n", fm.Original))
				sb.WriteString(fmt.Sprintf("           -> %s\n", fm.Migrated))
				for _, c := range fm.Changes {
					sb.WriteString(fmt.Sprintf("           (%s)\n", c))
				}
			}

			if !fm.Valid {
				sb.WriteString(fmt.Sprintf("    [ERROR] Invalid: %s\n", fm.Error))
			}
			sb.WriteString("\n")
		}
	}

	// Warnings
	if len(r.Warnings) > 0 {
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		sb.WriteString("WARNINGS\n")
		sb.WriteString(strings.Repeat("-", 80) + "\n\n")

		for _, w := range r.Warnings {
			sb.WriteString(fmt.Sprintf("  [WARN] %s\n", w.Message))
			if w.Path != "" {
				sb.WriteString(fmt.Sprintf("         Path: %s\n", w.Path))
			}
			if w.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("         Suggestion: %s\n", w.Suggestion))
			}
			sb.WriteString("\n")
		}
	}

	// Errors
	if len(r.Errors) > 0 {
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		sb.WriteString("ERRORS\n")
		sb.WriteString(strings.Repeat("-", 80) + "\n\n")

		for _, e := range r.Errors {
			severity := "[ERROR]"
			if e.Fatal {
				severity = "[FATAL]"
			}
			sb.WriteString(fmt.Sprintf("  %s %s\n", severity, e.Message))
			if e.Path != "" {
				sb.WriteString(fmt.Sprintf("         Path: %s\n", e.Path))
			}
			sb.WriteString("\n")
		}
	}

	// Template Issues
	if len(r.TemplateIssues) > 0 {
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		sb.WriteString("TEMPLATE ISSUES\n")
		sb.WriteString(strings.Repeat("-", 80) + "\n\n")

		for _, ti := range r.TemplateIssues {
			severity := strings.ToUpper(ti.Severity)
			sb.WriteString(fmt.Sprintf("  [%s] %s:%d\n", severity, ti.File, ti.Line))
			sb.WriteString(fmt.Sprintf("         %s\n", ti.Issue))
			if ti.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("         Suggestion: %s\n", ti.Suggestion))
			}
			sb.WriteString("\n")
		}
	}

	// Next Steps
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString("NEXT STEPS\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n\n")

	step := 1
	if len(r.Warnings) > 0 || len(r.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("  %d. Review the warnings and errors above\n", step))
		step++
	}

	if r.OutputFile != "" {
		sb.WriteString(fmt.Sprintf("  %d. Config written to: %s\n", step, r.OutputFile))
	} else {
		sb.WriteString(fmt.Sprintf("  %d. Run: markata-go migrate -o markata-go.toml\n", step))
	}
	step++

	if len(r.TemplateIssues) > 0 {
		sb.WriteString(fmt.Sprintf("  %d. Update templates as noted above\n", step))
		step++
	}

	sb.WriteString(fmt.Sprintf("  %d. Test with: markata-go build --dry-run\n", step))
	step++
	sb.WriteString(fmt.Sprintf("  %d. Full build: markata-go build\n", step))

	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	return sb.String()
}

// ShortReport generates a concise migration summary.
func (r *MigrationResult) ShortReport() string {
	var sb strings.Builder

	sb.WriteString("Migration Analysis:\n")
	sb.WriteString(fmt.Sprintf("  Config changes:    %d\n", len(r.Changes)))
	sb.WriteString(fmt.Sprintf("  Filter migrations: %d\n", len(r.FilterMigrations)))
	sb.WriteString(fmt.Sprintf("  Warnings:          %d\n", len(r.Warnings)))
	sb.WriteString(fmt.Sprintf("  Errors:            %d\n", len(r.Errors)))

	if r.HasErrors() {
		sb.WriteString("\nPlease resolve errors before migrating.\n")
	} else if r.HasWarnings() {
		sb.WriteString("\nReady to migrate (review warnings).\n")
	} else {
		sb.WriteString("\nReady to migrate!\n")
	}

	return sb.String()
}

// JSONReport returns a JSON-friendly structure for programmatic use.
func (r *MigrationResult) JSONReport() map[string]interface{} {
	return map[string]interface{}{
		"input_file":        r.InputFile,
		"output_file":       r.OutputFile,
		"timestamp":         r.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		"changes_count":     len(r.Changes),
		"migrations_count":  len(r.FilterMigrations),
		"warnings_count":    len(r.Warnings),
		"errors_count":      len(r.Errors),
		"exit_code":         r.ExitCode(),
		"has_errors":        r.HasErrors(),
		"has_warnings":      r.HasWarnings(),
		"changes":           r.Changes,
		"filter_migrations": r.FilterMigrations,
		"warnings":          r.Warnings,
		"errors":            r.Errors,
		"template_issues":   r.TemplateIssues,
	}
}
