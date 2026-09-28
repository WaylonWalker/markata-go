package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

const cleanPublishServeHeartbeatInterval = time.Second

func init() {
	originalRunE := serveCmd.RunE
	serveCmd.RunE = func(cmd *cobra.Command, args []string) error {
		finalOutput, err := resolveCleanBuildOutput()
		if err != nil {
			return originalRunE(cmd, args)
		}
		finalOutput, err = filepath.Abs(finalOutput)
		if err != nil {
			return originalRunE(cmd, args)
		}

		stopHeartbeat, err := startCleanPublishServeHeartbeat(finalOutput)
		if err != nil {
			warnf("could not advertise served output for clean-build handoff: %v", err)
			return originalRunE(cmd, args)
		}
		defer stopHeartbeat()
		return originalRunE(cmd, args)
	}
}

func startCleanPublishServeHeartbeat(output string) (func(), error) {
	dir := cleanPublishServeMarkerDir(output)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return func() {}, err
	}
	marker := filepath.Join(dir, fmt.Sprintf(".%d-%d", os.Getpid(), time.Now().UnixNano()))
	touch := func() error {
		now := time.Now()
		file, err := os.OpenFile(marker, os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
		return os.Chtimes(marker, now, now)
	}
	if err := touch(); err != nil {
		return func() {}, err
	}

	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		ticker := time.NewTicker(cleanPublishServeHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := touch(); err != nil && verbose {
					verbosef("serve clean-publish heartbeat failed: %v", err)
				}
			}
		}
	}()

	return func() {
		close(done)
		<-finished
		_ = os.Remove(marker)
		_ = os.Remove(dir)
	}, nil
}
