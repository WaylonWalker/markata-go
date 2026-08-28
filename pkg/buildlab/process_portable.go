//go:build !linux

package buildlab

import (
	"errors"
	"os/exec"
)

var errProcessGroupsUnsupported = errors.New("Build Lab process groups are unsupported on this platform")

// Build Lab requires a process-group implementation so that timeouts terminate
// descendant tools. Add a platform implementation before enabling it there.
func configureProcessGroup(*exec.Cmd) error { return errProcessGroupsUnsupported }
func killProcessGroup(int) error            { return errProcessGroupsUnsupported }
func killProcess(int) error                 { return errProcessGroupsUnsupported }
