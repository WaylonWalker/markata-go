package builderadmin

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

const buildSpaceReserveBytes = 64 << 20

func (s *Service) preflightBuildSpace(current string) error {
	free, supported, err := diskFreeBytes(s.cfg.SiteDir)
	if err != nil {
		return fmt.Errorf("check free space on site volume: %w", err)
	}
	if !supported {
		return nil
	}

	required := uint64(buildSpaceReserveBytes)
	if current != "" {
		size, err := directorySize(current)
		if err != nil {
			return fmt.Errorf("measure current release for free-space preflight: %w", err)
		}
		required += size
	}
	return checkFreeSpace(free, required)
}

func checkFreeSpace(available, required uint64) error {
	if available < required {
		return fmt.Errorf("insufficient free space for build candidate: need at least %d bytes, have %d bytes", required, available)
	}
	return nil
}

func directorySize(path string) (uint64, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return 0, err
	}
	var total uint64
	err = filepath.WalkDir(resolved, func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			total += uint64(info.Size())
		}
		return nil
	})
	return total, err
}
