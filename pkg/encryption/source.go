package encryption

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// SourceMarkerPrefix identifies Markdown bodies encrypted for safe storage.
	SourceMarkerPrefix = "<!-- markata-encrypted-source:v1"
	sourceMarkerEnd    = "-->"
)

var sourceMarkerPattern = regexp.MustCompile(`^<!--\s*markata-encrypted-source:v1(?:\s+key=([A-Za-z0-9_.-]+))?\s*-->\s*\n?([A-Za-z0-9+/=\s]+)$`)

// SourceEnvelope describes an encrypted Markdown source body.
type SourceEnvelope struct {
	KeyName    string
	Ciphertext string
}

// IsSourceEncrypted reports whether body starts with the encrypted source marker.
func IsSourceEncrypted(body string) bool {
	return strings.HasPrefix(strings.TrimSpace(body), SourceMarkerPrefix)
}

// ParseSourceEnvelope extracts the key name and ciphertext from an encrypted source body.
func ParseSourceEnvelope(body string) (SourceEnvelope, bool, error) {
	trimmed := strings.TrimSpace(body)
	if !strings.HasPrefix(trimmed, SourceMarkerPrefix) {
		return SourceEnvelope{}, false, nil
	}

	matches := sourceMarkerPattern.FindStringSubmatch(trimmed)
	if matches == nil {
		return SourceEnvelope{}, true, fmt.Errorf("encryption: malformed encrypted source marker")
	}

	ciphertext := strings.Join(strings.Fields(matches[2]), "")
	if ciphertext == "" {
		return SourceEnvelope{}, true, ErrEmptyCiphertext
	}

	return SourceEnvelope{
		KeyName:    matches[1],
		Ciphertext: ciphertext,
	}, true, nil
}

// FormatSourceEncrypted returns a Markdown body containing the encrypted source marker.
func FormatSourceEncrypted(ciphertext, keyName string) string {
	keyName = strings.TrimSpace(keyName)
	if keyName == "" {
		return SourceMarkerPrefix + " " + sourceMarkerEnd + "\n" + ciphertext + "\n"
	}
	return SourceMarkerPrefix + " key=" + keyName + " " + sourceMarkerEnd + "\n" + ciphertext + "\n"
}

// EncryptSourceMarkdown encrypts plaintext Markdown and wraps it in the source marker.
func EncryptSourceMarkdown(markdown, keyName, password string) (string, error) {
	ciphertext, err := Encrypt([]byte(markdown), password)
	if err != nil {
		return "", err
	}
	return FormatSourceEncrypted(ciphertext, keyName), nil
}

// DecryptSourceMarkdown decrypts a source-encrypted Markdown body.
func DecryptSourceMarkdown(body, password string) (markdown, keyName string, err error) {
	envelope, encrypted, err := ParseSourceEnvelope(body)
	if err != nil {
		return "", "", err
	}
	if !encrypted {
		return body, "", nil
	}
	plaintext, err := Decrypt(envelope.Ciphertext, password)
	if err != nil {
		return "", envelope.KeyName, err
	}
	return string(plaintext), envelope.KeyName, nil
}
