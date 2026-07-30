package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunExplain_UnknownTopicReturnsError(t *testing.T) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	explainCmd.SetOut(stdout)
	explainCmd.SetErr(stderr)

	err := runExplain(explainCmd, []string{"wat"})
	if err == nil {
		t.Fatal("expected error for unknown topic")
	}
	if !strings.Contains(err.Error(), "unknown topic \"wat\"") {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no direct stderr output, got %q", stderr.String())
	}
}

func TestRunExplain_ContentTopicDescribesAgentWorkflow(t *testing.T) {
	stdout := bytes.NewBuffer(nil)
	explainCmd.SetOut(stdout)

	if err := runExplain(explainCmd, []string{"content"}); err != nil {
		t.Fatalf("runExplain(content) error = %v", err)
	}
	for _, want := range []string{"--site-dir", "search", "--no-input", "edit"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("content explanation missing %q:\n%s", want, stdout.String())
		}
	}
}
