//go:build linux

package builderadmin

import (
	"io/fs"
	"syscall"
)

func publicationFileIdentity(info fs.FileInfo) publicationIdentity {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return publicationIdentity{}
	}
	return publicationIdentity{Device: stat.Dev, Inode: stat.Ino, Size: info.Size(), Mode: uint32(info.Mode()), Modified: info.ModTime().UnixNano(), Changed: stat.Ctim.Nano()}
}
