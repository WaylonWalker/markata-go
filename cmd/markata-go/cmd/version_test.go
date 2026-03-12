package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand_ShortUsesStdoutOnly(t *testing.T) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	versionCmd.SetOut(stdout)
	versionCmd.SetErr(stderr)
	currentCmd = versionCmd
	defer func() { currentCmd = nil }()

	if err := versionCmd.Flags().Set("short", "true"); err != nil {
		t.Fatalf("set short flag: %v", err)
	}
	defer func() {
		_ = versionCmd.Flags().Set("short", "false")
	}()

	versionCmd.Run(versionCmd, nil)

	if got := strings.TrimSpace(stdout.String()); got != GetVersion() {
		t.Fatalf("stdout = %q, want %q", got, GetVersion())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}
