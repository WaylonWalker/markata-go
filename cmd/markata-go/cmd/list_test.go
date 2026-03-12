package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderPostsTable_UsesCommandWriter(t *testing.T) {
	stdout := bytes.NewBuffer(nil)
	listCmd.SetOut(stdout)
	currentCmd = listCmd
	defer func() { currentCmd = nil }()

	err := renderPostsTable([]postRow{{
		Title:       "Hello",
		Date:        "2024-01-15",
		Words:       120,
		ReadingTime: 1,
		Tags:        []string{"go"},
		Path:        "posts/hello.md",
	}})
	if err != nil {
		t.Fatalf("renderPostsTable() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "TITLE") || !strings.Contains(output, "posts/hello.md") {
		t.Fatalf("expected table output in command writer, got %q", output)
	}
}
