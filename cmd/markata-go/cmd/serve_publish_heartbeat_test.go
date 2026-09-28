package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanPublishServeHeartbeatAdvertisesOutput(t *testing.T) {
	output := filepath.Join(t.TempDir(), "output")
	stop, err := startCleanPublishServeHeartbeat(output)
	if err != nil {
		t.Fatalf("start heartbeat: %v", err)
	}

	if !cleanPublishServeActive(output, time.Now()) {
		stop()
		t.Fatal("fresh serve heartbeat did not advertise output")
	}

	stop()
	if cleanPublishServeActive(output, time.Now()) {
		t.Fatal("stopped serve heartbeat still advertises output")
	}
}

func TestCleanPublishServeActiveIgnoresStaleMarkers(t *testing.T) {
	output := filepath.Join(t.TempDir(), "output")
	dir := cleanPublishServeMarkerDir(output)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "stale")
	if err := os.WriteFile(marker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().Add(-cleanPublishServeMarkerTTL - time.Second)
	if err := os.Chtimes(marker, stale, stale); err != nil {
		t.Fatal(err)
	}

	if cleanPublishServeActive(output, time.Now()) {
		t.Fatal("stale marker advertised output")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("stale marker was not pruned: %v", err)
	}
}
