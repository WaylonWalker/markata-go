package cmd

import (
	"bytes"
	"strings"
	"testing"

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
