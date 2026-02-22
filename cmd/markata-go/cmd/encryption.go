package cmd

import (
	"fmt"

	"github.com/WaylonWalker/markata-go/pkg/encryption"
	"github.com/spf13/cobra"
)

var (
	encryptionPasswordLength int
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

func init() {
	encryptionCmd.AddCommand(generatePasswordCmd)
	generatePasswordCmd.Flags().IntVar(&encryptionPasswordLength, "length", encryption.DefaultMinPasswordLength, "password length (must be at least the configured minimum)")
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
