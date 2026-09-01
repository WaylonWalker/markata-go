//go:build linux

package buildlab

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRun_TimeoutKillsDescendantProcessGroup(t *testing.T) {
	root := t.TempDir()
	pidPath := filepath.Join(root, "child.pid")
	quotedPIDPath := "'" + strings.ReplaceAll(pidPath, "'", "'\"'\"'") + "'"
	script := fmt.Sprintf("sleep 30 & child=$!; printf '%%s' \"$child\" > %s; wait", quotedPIDPath)

	started := time.Now()
	r := Run(context.Background(), RunConfig{
		Command: "sh",
		Args:    []string{"-c", script},
		CWD:     root,
		Timeout: 100 * time.Millisecond,
	})
	if !r.TimedOut {
		t.Fatalf("run was not timed out: %+v", r)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("timeout took too long: %v", elapsed)
	}

	var data []byte
	var err error
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		data, err = os.ReadFile(pidPath)
		if err == nil && len(data) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err != nil || len(data) == 0 {
		t.Fatalf("child pid was not recorded: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("child pid = %q: %v", data, err)
	}
	deadline = time.Now().Add(time.Second)
	for processIsAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if processIsAlive(pid) {
		// Avoid treating a process that exits immediately after the final
		// procfs read as a survivor.
		time.Sleep(20 * time.Millisecond)
		if processIsAlive(pid) {
			t.Fatalf("descendant process %d survived timeout", pid)
		}
	}
}

func TestRun_SuccessKillsBackgroundProcessGroup(t *testing.T) {
	root := t.TempDir()
	pidPath := filepath.Join(root, "child.pid")
	quotedPIDPath := "'" + strings.ReplaceAll(pidPath, "'", "'\"'\"'") + "'"
	script := fmt.Sprintf("sleep 30 >/dev/null 2>&1 & child=$!; printf '%%s' \"$child\" > %s; exit 0", quotedPIDPath)

	r := Run(context.Background(), RunConfig{
		Command: "sh",
		Args:    []string{"-c", script},
		CWD:     root,
		Timeout: 5 * time.Second,
	})
	if !r.Successful() {
		t.Fatalf("run was not successful: %+v", r)
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("read child pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("child pid = %q: %v", data, err)
	}
	deadline := time.Now().Add(time.Second)
	for processIsAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if processIsAlive(pid) {
		t.Fatalf("descendant process %d survived successful run cleanup", pid)
	}
}

func processIsAlive(pid int) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return false
	}
	end := strings.LastIndex(string(data), ") ")
	return end < 0 || end+2 >= len(data) || data[end+2] != 'Z'
}
