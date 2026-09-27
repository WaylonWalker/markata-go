//go:build windows

package servefix

import (
	"golang.org/x/sys/windows"
)

func replacePath(source, destination string) error {
	sourcePath, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	destinationPath, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	// MoveFileEx with REPLACE_EXISTING replaces the destination in one
	// filesystem operation, unlike os.Rename on Windows when the destination
	// already exists. The temp file is created beside the source, so this stays
	// on the same volume and needs no copy/delete fallback.
	return windows.MoveFileEx(sourcePath, destinationPath, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
