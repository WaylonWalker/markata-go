package models

import (
	"encoding/json"
	"strings"
)

// LicenseOption describes a supported license key and the human-readable
// attribution that is rendered in the footer.
type LicenseOption struct {
	Key         string
	Name        string
	URL         string
	Description string
	Recommended bool
}

// DefaultLicenseKey is the license key used by the init wizard when no choice is provided.
const DefaultLicenseKey = "cc-by-4.0"

// LicenseOptions lists the built-in license choices in display order.
var LicenseOptions = []LicenseOption{
	{Key: "all-rights-reserved", Name: "All rights reserved", Description: "No reuse without permission", URL: "", Recommended: false},
	{Key: "cc-by-4.0", Name: "Creative Commons Attribution 4.0", Description: "Reuse with attribution", URL: "https://creativecommons.org/licenses/by/4.0/", Recommended: true},
	{Key: "cc-by-sa-4.0", Name: "Creative Commons Attribution-ShareAlike 4.0", Description: "Reuse with attribution and share-alike", URL: "https://creativecommons.org/licenses/by-sa/4.0/", Recommended: false},
	{Key: "cc-by-nc-4.0", Name: "Creative Commons Attribution-NonCommercial 4.0", Description: "Reuse non-commercially with attribution", URL: "https://creativecommons.org/licenses/by-nc/4.0/", Recommended: false},
	{Key: "cc-by-nd-4.0", Name: "Creative Commons Attribution-NoDerivatives 4.0", Description: "Reuse with attribution, no derivatives", URL: "https://creativecommons.org/licenses/by-nd/4.0/", Recommended: false},
	{Key: "cc-by-nc-sa-4.0", Name: "Creative Commons Attribution-NonCommercial-ShareAlike 4.0", Description: "Non-commercial reuse with attribution and share-alike", URL: "https://creativecommons.org/licenses/by-nc-sa/4.0/", Recommended: false},
	{Key: "mit", Name: "MIT License", Description: "Permissive open source license", URL: "https://opensource.org/licenses/MIT", Recommended: false},
}

var licenseLookup = buildLicenseLookup()

func buildLicenseLookup() map[string]LicenseOption {
	lookup := make(map[string]LicenseOption, len(LicenseOptions))
	for _, opt := range LicenseOptions {
		lookup[strings.ToLower(opt.Key)] = opt
	}
	return lookup
}

// LicenseKeys returns the supported license keys in display order.
func LicenseKeys() []string {
	keys := make([]string, len(LicenseOptions))
	for i, opt := range LicenseOptions {
		keys[i] = opt.Key
	}
	return keys
}

// GetLicenseOption looks up the license option for a normalized key.
func GetLicenseOption(key string) (LicenseOption, bool) {
	opt, ok := licenseLookup[strings.ToLower(strings.TrimSpace(key))]
	return opt, ok
}

// LicenseValue tracks the raw value provided for the `license` key.
// It can be either a string, or the boolean false when the setting is disabled.
type LicenseValue struct {
	Raw interface{}
}

// HasValue reports whether any explicit license value was provided.
func (l LicenseValue) HasValue() bool {
	return l.Raw != nil
}

// IsDisabled returns true when the value was the boolean false starter.
func (l LicenseValue) IsDisabled() bool {
	if val, ok := l.Raw.(bool); ok {
		return !val
	}
	return false
}

// Key returns the normalized string key when the value is a string.
func (l LicenseValue) Key() (string, bool) {
	if s, ok := l.Raw.(string); ok {
		return strings.ToLower(strings.TrimSpace(s)), s != ""
	}
	return "", false
}

// MarshalJSON writes the raw value so serialized configs match the input.
func (l LicenseValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.Raw)
}

// MarshalTOML implements the BurntSushi toml.Marshaler interface.
func (l LicenseValue) MarshalTOML() (interface{}, error) {
	return l.Raw, nil
}

// MarshalYAML implements yaml.Marshaler.
func (l LicenseValue) MarshalYAML() (interface{}, error) {
	return l.Raw, nil
}
