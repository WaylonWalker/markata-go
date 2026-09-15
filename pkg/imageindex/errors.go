package imageindex

import "errors"

// ErrUnsupportedVersion indicates that an artifact uses a newer image-index
// generation than this package understands.
var ErrUnsupportedVersion = errors.New("unsupported image index version")
