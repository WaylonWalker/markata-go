package buildlab

import "os/exec"

// processTree owns the lifetime of one Build Lab command and its descendants.
// Start must not return until the command is contained by the platform's
// process-group or job mechanism. This is lifecycle cleanup, not a security
// sandbox.
type processTree interface {
	Start(*exec.Cmd) error
	Kill() error
	Close() error
}
