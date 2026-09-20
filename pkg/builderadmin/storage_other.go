//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd

package builderadmin

func diskFreeBytes(_ string) (uint64, bool, error) {
	// The copy command remains strict on platforms without a portable statfs
	// implementation. The preflight is an additional safeguard, not the only
	// protection against a partial candidate.
	return 0, false, nil
}
