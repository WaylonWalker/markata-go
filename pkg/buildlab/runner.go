package buildlab

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// DefaultMaxOutputBytes bounds each child stream retained in a result.
const DefaultMaxOutputBytes = 8 << 20

type RunConfig struct {
	Command        string
	Args           []string
	CWD            string
	Env            []string
	Timeout        time.Duration
	MaxOutputBytes int
}
type RunResult struct {
	Stdout, Stderr  []byte
	ExitCode        int
	Duration        time.Duration
	TimedOut        bool
	StdoutTruncated bool
	StderrTruncated bool
	Err             error
}

const processWaitDelay = time.Second

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
	cmd := exec.CommandContext(cctx, cfg.Command, cfg.Args...)
	// waitForProcess owns process-tree cancellation. WaitDelay still bounds
	// reaping if a child exits while a descendant retains an output pipe.
	cmd.Cancel = nil
	cmd.WaitDelay = processWaitDelay
	cmd.Dir = cfg.CWD
	if cfg.Env != nil {
		cmd.Env = cfg.Env
	}
	limit := cfg.MaxOutputBytes
	if limit == 0 {
		limit = DefaultMaxOutputBytes
	}
	if limit < 0 {
		return RunResult{Duration: time.Since(start), Err: fmt.Errorf("max output bytes must not be negative")}
	}
	var out, errOut limitedBuffer
	out.limit = limit
	errOut.limit = limit
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	tree, err := newProcessTree(cmd)
	if err != nil {
		return RunResult{Stdout: out.Bytes(), Stderr: errOut.Bytes(), StdoutTruncated: out.truncated, StderrTruncated: errOut.truncated, Duration: time.Since(start), Err: err}
	}
	err = tree.Start(cmd)
	if err != nil {
		r := RunResult{Stdout: out.Bytes(), Stderr: errOut.Bytes(), StdoutTruncated: out.truncated, StderrTruncated: errOut.truncated, Duration: time.Since(start), Err: errors.Join(err, tree.Close())}
		if cmd.ProcessState != nil {
			r.ExitCode = cmd.ProcessState.ExitCode()
		}
		return r
	}
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	err = waitForProcess(cctx, wait, tree)
	if errors.Is(err, exec.ErrWaitDelay) {
		// A detached descendant can keep output pipes open after the direct
		// process exits. Terminate the process tree before returning a failed
		// observation so it cannot outlive the Build Lab workspace.
		err = errors.Join(err, tree.Kill())
	}
	err = errors.Join(err, tree.Close())
	if out.truncated {
		err = errors.Join(err, fmt.Errorf("stdout exceeded %d-byte limit", limit))
	}
	if errOut.truncated {
		err = errors.Join(err, fmt.Errorf("stderr exceeded %d-byte limit", limit))
	}
	r := RunResult{
		Stdout: out.Bytes(), Stderr: errOut.Bytes(), Duration: time.Since(start), Err: err,
		TimedOut:        errors.Is(cctx.Err(), context.DeadlineExceeded),
		StdoutTruncated: out.truncated, StderrTruncated: errOut.truncated,
	}
	if cmd.ProcessState != nil {
		r.ExitCode = cmd.ProcessState.ExitCode()
	}
	return r
}

type limitedBuffer struct {
	data      []byte
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - len(b.data)
	if remaining > 0 {
		if remaining > len(p) {
			remaining = len(p)
		}
		b.data = append(b.data, p[:remaining]...)
	}
	if len(p) > remaining {
		b.truncated = true
	}
	return len(p), nil
}

func (b *limitedBuffer) Bytes() []byte { return b.data }

func waitForProcess(ctx context.Context, wait <-chan error, tree processTree) error {
	select {
	case err := <-wait:
		return err
	case <-ctx.Done():
		select {
		case err := <-wait:
			return err
		default:
		}
		// Kill the complete process group before waiting for the direct child.
		// This closes inherited output descriptors held by descendant tools and
		// prevents cmd.Wait from blocking on those descendants.
		killErr := tree.Kill()
		select {
		case waitErr := <-wait:
			return errors.Join(ctx.Err(), killErr, waitErr)
		case <-time.After(2 * processWaitDelay):
			return errors.Join(ctx.Err(), killErr, fmt.Errorf("waiting for terminated build process exceeded %s", 2*processWaitDelay))
		}
	}
}

func (r RunResult) Successful() bool { return r.Err == nil && !r.TimedOut && r.ExitCode == 0 }
