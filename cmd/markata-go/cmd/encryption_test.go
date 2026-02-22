package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/encryption"
)

func TestGeneratePasswordCommand_DefaultLength(t *testing.T) {
	originalLength := encryptionPasswordLength
	defer func() { encryptionPasswordLength = originalLength }()
	encryptionPasswordLength = encryption.DefaultMinPasswordLength

	buf := bytes.NewBuffer(nil)
	generatePasswordCmd.SetOut(buf)

	if err := runGeneratePasswordCommand(generatePasswordCmd, nil); err != nil {
		t.Fatalf("runGeneratePasswordCommand() error = %v", err)
	}

	password := strings.TrimSpace(buf.String())
	if len(password) != encryption.DefaultMinPasswordLength {
		t.Errorf("password length = %d, want %d", len(password), encryption.DefaultMinPasswordLength)
	}
	if err := encryption.ValidatePassword(password, encryption.DefaultMinPasswordLength, encryption.DefaultMinEstimatedCrackDuration); err != nil {
		t.Fatalf("generated password failed validation: %v", err)
	}
}

func TestGeneratePasswordCommand_LengthTooShort(t *testing.T) {
	originalLength := encryptionPasswordLength
	defer func() { encryptionPasswordLength = originalLength }()
	encryptionPasswordLength = encryption.DefaultMinPasswordLength - 1

	buf := bytes.NewBuffer(nil)
	generatePasswordCmd.SetOut(buf)

	if err := runGeneratePasswordCommand(generatePasswordCmd, nil); err == nil {
		t.Error("expected error when requested length < minimum")
	}
}

func TestCheckPasswordCommand_Pass(t *testing.T) {
	configPath := writeEncryptionConfigFile(t)
	originalCfg := cfgFile
	originalKey := encryptionCheckKey
	defer func() {
		cfgFile = originalCfg
		encryptionCheckKey = originalKey
	}()

	cfgFile = configPath
	encryptionCheckKey = ""
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_DEFAULT", "Safe-Passphrase-2026!")

	buf := bytes.NewBuffer(nil)
	checkPasswordCmd.SetOut(buf)

	if err := runCheckPasswordCommand(checkPasswordCmd, nil); err != nil {
		t.Fatalf("runCheckPasswordCommand() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "PASS default") {
		t.Fatalf("expected PASS output, got %q", output)
	}
}

func TestCheckPasswordCommand_Fail(t *testing.T) {
	configPath := writeEncryptionConfigFile(t)
	originalCfg := cfgFile
	originalKey := encryptionCheckKey
	defer func() {
		cfgFile = originalCfg
		encryptionCheckKey = originalKey
	}()

	cfgFile = configPath
	encryptionCheckKey = ""
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_DEFAULT", "weak")

	buf := bytes.NewBuffer(nil)
	checkPasswordCmd.SetOut(buf)

	err := runCheckPasswordCommand(checkPasswordCmd, nil)
	if err == nil {
		t.Fatal("expected runCheckPasswordCommand() to fail")
	}

	output := buf.String()
	if !strings.Contains(output, "FAIL default") {
		t.Fatalf("expected FAIL output, got %q", output)
	}
}

func writeEncryptionConfigFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "markata-go.toml")
	content := `[markata-go]
title = "test"

[markata-go.encryption]
enabled = true
default_key = "default"
enforce_strength = true
min_password_length = 14
min_estimated_crack_time = "10y"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestFormatCrackDurationHuman(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "years", in: 10 * 365 * 24 * time.Hour, want: "10.0y"},
		{name: "days", in: 48 * time.Hour, want: "2.0d"},
		{name: "hours", in: 90 * time.Minute, want: "1.5h"},
		{name: "zero", in: 0, want: "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCrackDurationHuman(tt.in)
			if got != tt.want {
				t.Fatalf("formatCrackDurationHuman(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
