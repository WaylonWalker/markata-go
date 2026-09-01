//go:build !linux && !windows && !darwin && !dragonfly && !freebsd && !netbsd && !openbsd && !solaris

package buildlab

import (
	"errors"
	"os/exec"
)

var errProcessGroupsUnsupported = errors.New("Build Lab process groups are unsupported on this platform")

// Build Lab requires a process-group implementation so that timeouts terminate
// descendant tools. Add a platform implementation before enabling it there.
func newProcessTree(*exec.Cmd) (processTree, error) { return nil, errProcessGroupsUnsupported }
