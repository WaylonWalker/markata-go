package builderadmin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const prunedReleasePrefix = ".pruning-"

func (s *Service) pruneReleases() error {
	return s.pruneReleasesWithRemove(os.RemoveAll)
}

// pruneReleasesWithRemove detaches obsolete releases under the publication lock,
// then deletes their trees without delaying promotion or rollback.
func (s *Service) pruneReleasesWithRemove(removeTree func(string) error) error {
	releasesDir := filepath.Join(s.cfg.SiteDir, "releases")
	entries, err := os.ReadDir(releasesDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("list detached releases: %w", err)
	}
	var errs []error
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), prunedReleasePrefix) {
			continue
		}
		if entry.Name() == s.currentReleaseID() {
			continue
		}
		if err := removeTree(filepath.Join(releasesDir, entry.Name())); err != nil {
			errs = append(errs, fmt.Errorf("delete detached release %s: %w", entry.Name(), err))
		}
	}
	releases := s.discoverReleases()
	if len(releases) <= s.cfg.ReleasesKeep {
		return errors.Join(errs...)
	}
	for _, release := range releases[s.cfg.ReleasesKeep:] {
		s.releaseMu.Lock()
		if release.Current || release.ID == s.currentReleaseID() {
			s.releaseMu.Unlock()
			continue
		}
		detached := filepath.Join(releasesDir, prunedReleasePrefix+release.ID)
		err := os.Rename(release.Path, detached)
		s.releaseMu.Unlock()
		if err != nil {
			errs = append(errs, fmt.Errorf("detach release %s: %w", release.ID, err))
			continue
		}
		if err := removeTree(detached); err != nil {
			errs = append(errs, fmt.Errorf("delete detached release %s: %w", release.ID, err))
		}
	}
	return errors.Join(errs...)
}
