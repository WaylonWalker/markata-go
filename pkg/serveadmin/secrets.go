package serveadmin

import (
	"errors"
	"os"
	"path/filepath"
)

var (
	ErrNoSecrets      = errors.New("no secrets found")
	ErrInvalidSecrets = errors.New("invalid secrets")
)

type Secrets struct {
	Dir           string
	AdminUsername string
	AdminPassword string // hashed
	SessionKey    string
}

// DefaultSecretsDir is the default directory for admin secrets
const DefaultSecretsDir = ".markata-secrets"

// LoadSecrets loads admin secrets from the specified directory
func LoadSecrets(dir string) (*Secrets, error) {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSecrets
		}
		return nil, err
	}

	if !info.IsDir() {
		return nil, ErrInvalidSecrets
	}

	s := &Secrets{Dir: dir}

	// Load admin username
	usernamePath := filepath.Join(dir, "admin_username")
	if data, err := os.ReadFile(usernamePath); err == nil {
		s.AdminUsername = string(data)
	} else {
		return nil, ErrInvalidSecrets
	}

	// Load admin password hash
	passwordPath := filepath.Join(dir, "admin_password_hash")
	if data, err := os.ReadFile(passwordPath); err == nil {
		s.AdminPassword = string(data)
	} else {
		return nil, ErrInvalidSecrets
	}

	// Load session key
	keyPath := filepath.Join(dir, "session_hmac_key")
	if data, err := os.ReadFile(keyPath); err == nil {
		s.SessionKey = string(data)
	} else {
		return nil, ErrInvalidSecrets
	}

	return s, nil
}

// Exists checks if secrets exist in the directory
func SecretsExist(dir string) bool {
	_, err := LoadSecrets(dir)
	return err == nil
}

// CreateSecrets writes admin secrets to the directory
func CreateSecrets(dir, username, passwordHash, sessionKey string) error {
	// Create directory if needed
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	// Write each secret file
	files := map[string]string{
		"admin_username":      username,
		"admin_password_hash": passwordHash,
		"session_hmac_key":    sessionKey,
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return err
		}
	}

	return nil
}

// GetSecretsDir returns the configured secrets directory
func GetSecretsDir() string {
	// TODO: check config/env override
	return DefaultSecretsDir
}

// HasSecrets returns true if admin secrets exist
func HasSecrets() bool {
	return SecretsExist(GetSecretsDir())
}
