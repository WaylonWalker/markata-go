package buildlab

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

type RunConfig struct {
	Command string
	Args    []string
	CWD     string
	Env     []string
	Timeout time.Duration
}
type RunResult struct {
	Stdout, Stderr []byte
	ExitCode       int
	Duration       time.Duration
	TimedOut       bool
	Err            error
}

// Run starts a command in CWD and never changes the caller's directory. Env is
// an explicit environment; nil inherits the current environment.
func Run(ctx context.Context, cfg RunConfig) RunResult {
	start := time.Now()
	if ctx == nil {
		ctx = context.Background()
	}
	cctx := ctx
	cancel := func() {}
	if cfg.Timeout > 0 {
		cctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	}
	defer cancel()
	//nolint:gosec // Build Lab intentionally runs the configured build command.
	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Dir = cfg.CWD
	if cfg.Env != nil {
		cmd.Env = cfg.Env
	}
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := configureProcessGroup(cmd); err != nil {
		return RunResult{Stdout: out.Bytes(), Stderr: errOut.Bytes(), Duration: time.Since(start), Err: err}
	}
	err := cmd.Start()
	if err != nil {
		return RunResult{Stdout: out.Bytes(), Stderr: errOut.Bytes(), Duration: time.Since(start), Err: err}
	}
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	err = waitForProcess(cctx, wait, cmd.Process.Pid)
	r := RunResult{
		Stdout: out.Bytes(), Stderr: errOut.Bytes(), Duration: time.Since(start), Err: err,
		TimedOut: errors.Is(cctx.Err(), context.DeadlineExceeded),
	}
	if cmd.ProcessState != nil {
		r.ExitCode = cmd.ProcessState.ExitCode()
	}
	return r
}

func waitForProcess(ctx context.Context, wait <-chan error, pid int) error {
	select {
	case err := <-wait:
		return err
	case <-ctx.Done():
		// Kill the complete process group before waiting for the direct child.
		// This closes inherited output descriptors held by descendant tools and
		// prevents cmd.Wait from blocking on those descendants.
		killErr := killProcessGroup(pid)
		if killErr != nil {
			// Keep a direct-child fallback for a process that exits while the
			// group signal is in flight.
			killErr = killProcess(pid)
		}
		<-wait
		if killErr != nil {
			return errors.Join(ctx.Err(), killErr)
		}
		return ctx.Err()
	}
}

func (r RunResult) Successful() bool { return r.Err == nil && !r.TimedOut && r.ExitCode == 0 }
