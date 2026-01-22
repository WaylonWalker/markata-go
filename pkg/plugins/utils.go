package plugins

import "regexp"

// slugifyRegex matches characters that are not alphanumeric, hyphens, or underscores.
// Used for generating URL-safe slugs.
var slugifyRegex = regexp.MustCompile(`[^a-z0-9\-_]+`)

// multiHyphenRegex matches multiple consecutive hyphens.
// Used for collapsing multiple hyphens into one during slug generation.
var multiHyphenRegex = regexp.MustCompile(`-+`)
