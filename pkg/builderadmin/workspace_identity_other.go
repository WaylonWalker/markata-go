//go:build !linux

package builderadmin

import "io/fs"

func publicationFileIdentity(_ fs.FileInfo) publicationIdentity { return publicationIdentity{} }
