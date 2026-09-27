//go:build windows

package servefix

import (
	"golang.org/x/sys/windows"
)

func replacePath(source, destination string) error {
	security, err := windows.GetNamedSecurityInfo(destination, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	acl, _, err := security.DACL()
	if err != nil {
		return err
	}
	if err := windows.SetNamedSecurityInfo(source, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		return err
	}

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
