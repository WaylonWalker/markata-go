package cmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/suggest"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type inputError struct {
	problem string
	matches []string
	prefix  string
	next    string
}

func (e *inputError) Error() string {
	var b strings.Builder
	b.WriteString(e.problem)
	if len(e.matches) > 0 {
		b.WriteString("\nDid you mean:")
		for _, match := range e.matches {
			fmt.Fprintf(&b, "\n  %s%s", e.prefix, match)
		}
	} else {
		b.WriteString("\nmarkata-go cannot infer what you intended.")
	}
	if e.next != "" {
		fmt.Fprintf(&b, "\nNext: %s", e.next)
	}
	return b.String()
}

func clarifyFlagError(cmd *cobra.Command, err error) error {
	message := err.Error()
	name, isUnknown := strings.CutPrefix(message, "unknown flag: --")
	if isUnknown {
		name = strings.TrimSpace(name)
		candidates := make([]string, 0)
		addFlags := func(flags *pflag.FlagSet) {
			flags.VisitAll(func(flag *pflag.Flag) {
				if !flag.Hidden && flag.Deprecated == "" {
					candidates = append(candidates, flag.Name)
				}
			})
		}
		addFlags(cmd.Flags())
		addFlags(cmd.InheritedFlags())
		return &inputError{
			problem: message,
			matches: suggest.Closest(name, candidates, 3),
			prefix:  "--",
			next:    "Run '" + cmd.CommandPath() + " --help' to see valid flags.",
		}
	}
	return &inputError{problem: message, next: "Run '" + cmd.CommandPath() + " --help' to check flag syntax."}
}

// FormatError renders a single fatal diagnostic for the command entry point.
// It preserves plain text when stderr is redirected or coloring is disabled.
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return ""
	}
	var input *inputError
	if errors.As(err, &input) {
		return styleErrorLines(message)
	}
	if !strings.Contains(message, "\nNext:") {
		path := rootCmd.CommandPath()
		if currentCmd != nil {
			path = currentCmd.CommandPath()
		}
		message += "\nNext: Run '" + path + " --help' for usage, or rerun with --verbose for more detail."
	}
	return styleErrorLines(message)
}

func styleErrorLines(message string) string {
	parts := strings.Split(message, "\n")
	theme := currentLogTheme
	enabled := colorEnabledFor(errorOutputIsTerminal())
	parts[0] = colorize("Error", theme.Error, enabled) + ": " + parts[0]
	for i := 1; i < len(parts); i++ {
		line := parts[i]
		switch {
		case line == "Did you mean:":
			parts[i] = colorize(line, theme.Component, enabled)
		case strings.HasPrefix(line, "  --") || strings.HasPrefix(line, "  markata-go "):
			parts[i] = colorize(line, theme.Success, enabled)
		case strings.HasPrefix(line, "Next:"):
			parts[i] = colorize("Next", theme.Component, enabled) + strings.TrimPrefix(line, "Next")
		}
	}
	return strings.Join(parts, "\n")
}

func configKeyError(key string) error {
	candidates := make([]string, 0)
	for _, field := range configSettingsForSuggestions() {
		if field == key {
			return &inputError{
				problem: fmt.Sprintf("configuration key %q is not set in this file", key),
				next:    "Run 'markata-go config show' to see its resolved value, including defaults.",
			}
		}
		candidates = append(candidates, field)
	}
	sort.Strings(candidates)
	return &inputError{
		problem: fmt.Sprintf("unknown configuration key %q", key),
		matches: suggest.Closest(key, candidates, 3),
		next:    "Run 'markata-go config show' to inspect the active configuration.",
	}
}
