//go:build !windows

package servefix

import "os"

func replacePath(source, destination string) error {
	return os.Rename(source, destination)
}
