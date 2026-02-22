package cmd

import (
	"fmt"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/config"
	"github.com/WaylonWalker/markata-go/pkg/encryption"
	"github.com/spf13/cobra"
)

var (
	encryptionPasswordLength int
	encryptionCheckKey       string
)

var encryptionCmd = &cobra.Command{
	Use:   "encryption",
	Short: "Utilities for encryption keys and passwords",
	Long: `Encryption utilities help you manage passwords for private posts.

Currently the only helper is a password generator that creates a policy-compliant password.
`,
}

var generatePasswordCmd = &cobra.Command{
	Use:     "generate-password",
	Aliases: []string{"gen"},
	Short:   "Generate a policy-compliant encryption password",
	Long: `Generate a password that satisfies the default encryption policy (>=14 chars, strong entropy).

The password is printed to stdout only, making it easy to pipe into secret stores or copy it from your terminal.
`,
	Args: cobra.NoArgs,
	RunE: runGeneratePasswordCommand,
}

var checkPasswordCmd = &cobra.Command{
	Use:   "check",
	Short: "Check configured encryption key strength",
	Long: `Check configured encryption keys against the active policy.

By default this checks every key required by your config (default_key and private_tags mappings).
Use --key to check one specific key name.
`,
	Args: cobra.NoArgs,
	RunE: runCheckPasswordCommand,
}

func init() {
	encryptionCmd.AddCommand(generatePasswordCmd, checkPasswordCmd)
	generatePasswordCmd.Flags().IntVar(&encryptionPasswordLength, "length", encryption.DefaultMinPasswordLength, "password length (must be at least the configured minimum)")
	checkPasswordCmd.Flags().StringVar(&encryptionCheckKey, "key", "", "specific key name to check (default: all required keys)")
	rootCmd.AddCommand(encryptionCmd)
}

func runGeneratePasswordCommand(cmd *cobra.Command, _ []string) error {
	password, err := encryption.GeneratePassword(encryptionPasswordLength)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), password)
	return nil
}

func runCheckPasswordCommand(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	results, minDuration, minLength, err := evaluateEncryptionKeyPolicy(cfg, encryptionCheckKey)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No encryption keys configured to check.")
		return nil
	}

	failures := 0
	fmt.Fprintf(cmd.OutOrStdout(), "Policy: min_length=%d, min_estimated_crack_time=%s\n", minLength, formatCrackDurationHuman(minDuration))
	for _, result := range results {
		if result.Err != nil {
			failures++
			if result.Configured {
				fmt.Fprintf(cmd.OutOrStdout(), "FAIL %s (%s): %s (estimated=%s)\n", result.KeyName, result.EnvName, result.Err, formatCrackDurationHuman(result.Estimated))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "FAIL %s (%s): %s\n", result.KeyName, result.EnvName, result.Err)
			}
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "PASS %s (%s): estimated=%s\n", result.KeyName, result.EnvName, formatCrackDurationHuman(result.Estimated))
	}

	if failures > 0 {
		return fmt.Errorf("%d encryption key(s) failed policy", failures)
	}

	if !cfg.Encryption.EnforceStrength {
		fmt.Fprintln(cmd.OutOrStdout(), "Warning: encryption.enforce_strength=false, builds will not enforce this policy.")
	}

	return nil
}

func formatCrackDurationHuman(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	if d < time.Second {
		return "<1s"
	}

	const year = 365 * 24 * time.Hour
	if d >= year {
		years := float64(d) / float64(year)
		if years >= 100 {
			return fmt.Sprintf("%.0fy", years)
		}
		return fmt.Sprintf("%.1fy", years)
	}

	const day = 24 * time.Hour
	if d >= day {
		days := float64(d) / float64(day)
		return fmt.Sprintf("%.1fd", days)
	}

	if d >= time.Hour {
		hours := float64(d) / float64(time.Hour)
		return fmt.Sprintf("%.1fh", hours)
	}

	return d.Round(time.Second).String()
}
