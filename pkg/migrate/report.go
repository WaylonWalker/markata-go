package migrate

import (
	"fmt"
	"strings"
)

// Report generates a human-readable migration report.
func (r *MigrationResult) Report() string {
	var sb strings.Builder

	r.writeHeader(&sb)
	r.writeSummary(&sb)
	r.writeConfigChanges(&sb)
	r.writeFilterMigrations(&sb)
	r.writeWarnings(&sb)
	r.writeErrors(&sb)
	r.writeTemplateIssues(&sb)
	r.writeNextSteps(&sb)

	sb.WriteString(strings.Repeat("=", 80) + "\n")

	return sb.String()
}

// writeHeader writes the report header section.
func (r *MigrationResult) writeHeader(sb *strings.Builder) {
	sb.WriteString(strings.Repeat("=", 80) + "\n")
	sb.WriteString("                        markata-go Migration Report\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")

	if r.InputFile != "" {
		fmt.Fprintf(sb, "Configuration File: %s\n", r.InputFile)
	}
	fmt.Fprintf(sb, "Generated: %s\n\n", r.Timestamp.Format("2006-01-02 15:04:05"))
}

// writeSummary writes the summary section.
func (r *MigrationResult) writeSummary(sb *strings.Builder) {
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString("SUMMARY\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n\n")

	status := "Ready to migrate"
	if r.HasErrors() {
		status = "Migration has issues"
	} else if r.HasWarnings() {
		status = "Ready to migrate (with warnings)"
	}
	fmt.Fprintf(sb, "  Status: %s\n\n", status)

	fmt.Fprintf(sb, "  Changes required:    %d\n", len(r.Changes))
	fmt.Fprintf(sb, "  Filter migrations:   %d\n", len(r.FilterMigrations))
	fmt.Fprintf(sb, "  Warnings:            %d\n", len(r.Warnings))
	fmt.Fprintf(sb, "  Errors:              %d\n", len(r.Errors))

	if len(r.TemplateIssues) > 0 {
		fmt.Fprintf(sb, "  Template issues:     %d\n", len(r.TemplateIssues))
	}

	sb.WriteString("\n")
}

// writeConfigChanges writes the configuration changes section.
func (r *MigrationResult) writeConfigChanges(sb *strings.Builder) {
	if len(r.Changes) == 0 {
		return
	}

	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString("CONFIGURATION CHANGES\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n\n")

	for _, change := range r.Changes {
		icon := "[MIGRATE]"
		fmt.Fprintf(sb, "  %s %s\n", icon, change.Description)
	}
	sb.WriteString("\n")
}

// writeFilterMigrations writes the filter migrations section.
func (r *MigrationResult) writeFilterMigrations(sb *strings.Builder) {
	if len(r.FilterMigrations) == 0 {
		return
	}

	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString("FILTER MIGRATIONS\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n\n")

	for _, fm := range r.FilterMigrations {
		feedName := fm.Feed
		if feedName == "" {
			feedName = "(unnamed)"
		}
		fmt.Fprintf(sb, "  Feed: %s\n", feedName)

		if len(fm.Changes) == 0 {
			fmt.Fprintf(sb, "    [OK] %s (no changes needed)\n", fm.Original)
		} else {
			fmt.Fprintf(sb, "    [MIGRATE] %s\n", fm.Original)
			fmt.Fprintf(sb, "           -> %s\n", fm.Migrated)
			for _, c := range fm.Changes {
				fmt.Fprintf(sb, "           (%s)\n", c)
			}
		}

		if !fm.Valid {
			fmt.Fprintf(sb, "    [ERROR] Invalid: %s\n", fm.Error)
		}
		sb.WriteString("\n")
	}
}

// writeWarnings writes the warnings section.
func (r *MigrationResult) writeWarnings(sb *strings.Builder) {
	if len(r.Warnings) == 0 {
		return
	}

	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString("WARNINGS\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n\n")

	for _, w := range r.Warnings {
		fmt.Fprintf(sb, "  [WARN] %s\n", w.Message)
		if w.Path != "" {
			fmt.Fprintf(sb, "         Path: %s\n", w.Path)
		}
		if w.Suggestion != "" {
			fmt.Fprintf(sb, "         Suggestion: %s\n", w.Suggestion)
		}
		sb.WriteString("\n")
	}
}

// writeErrors writes the errors section.
func (r *MigrationResult) writeErrors(sb *strings.Builder) {
	if len(r.Errors) == 0 {
		return
	}

	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString("ERRORS\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n\n")

	for _, e := range r.Errors {
		severity := "[ERROR]"
		if e.Fatal {
			severity = "[FATAL]"
		}
		fmt.Fprintf(sb, "  %s %s\n", severity, e.Message)
		if e.Path != "" {
			fmt.Fprintf(sb, "         Path: %s\n", e.Path)
		}
		sb.WriteString("\n")
	}
}

// writeTemplateIssues writes the template issues section.
func (r *MigrationResult) writeTemplateIssues(sb *strings.Builder) {
	if len(r.TemplateIssues) == 0 {
		return
	}

	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString("TEMPLATE ISSUES\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n\n")

	for _, ti := range r.TemplateIssues {
		severity := strings.ToUpper(ti.Severity)
		fmt.Fprintf(sb, "  [%s] %s:%d\n", severity, ti.File, ti.Line)
		fmt.Fprintf(sb, "         %s\n", ti.Issue)
		if ti.Suggestion != "" {
			fmt.Fprintf(sb, "         Suggestion: %s\n", ti.Suggestion)
		}
		sb.WriteString("\n")
	}
}

// writeNextSteps writes the next steps section.
func (r *MigrationResult) writeNextSteps(sb *strings.Builder) {
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString("NEXT STEPS\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n\n")

	step := 1
	if len(r.Warnings) > 0 || len(r.Errors) > 0 {
		fmt.Fprintf(sb, "  %d. Review the warnings and errors above\n", step)
		step++
	}

	if r.OutputFile != "" {
		fmt.Fprintf(sb, "  %d. Config written to: %s\n", step, r.OutputFile)
	} else {
		fmt.Fprintf(sb, "  %d. Run: markata-go migrate -o markata-go.toml\n", step)
	}
	step++

	if len(r.TemplateIssues) > 0 {
		fmt.Fprintf(sb, "  %d. Update templates as noted above\n", step)
		step++
	}

	fmt.Fprintf(sb, "  %d. Test with: markata-go build --dry-run\n", step)
	step++
	fmt.Fprintf(sb, "  %d. Full build: markata-go build\n", step)

	sb.WriteString("\n")
}

// ShortReport generates a concise migration summary.
func (r *MigrationResult) ShortReport() string {
	var sb strings.Builder

	sb.WriteString("Migration Analysis:\n")
	sb.WriteString(fmt.Sprintf("  Config changes:    %d\n", len(r.Changes)))
	sb.WriteString(fmt.Sprintf("  Filter migrations: %d\n", len(r.FilterMigrations)))
	sb.WriteString(fmt.Sprintf("  Warnings:          %d\n", len(r.Warnings)))
	sb.WriteString(fmt.Sprintf("  Errors:            %d\n", len(r.Errors)))

	switch {
	case r.HasErrors():
		sb.WriteString("\nPlease resolve errors before migrating.\n")
	case r.HasWarnings():
		sb.WriteString("\nReady to migrate (review warnings).\n")
	default:
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
