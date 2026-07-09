package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/encryption"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/plugins"
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
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_DEFAULT", "h7Qm!2Vx9#Lp4@Td")

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
	if !strings.Contains(output, "estimated=") {
		t.Fatalf("expected estimated crack time in fail output, got %q", output)
	}
}

func TestEncryptPostSourceFile_PrivatePost(t *testing.T) {
	cfg := testEncryptPostsConfig()
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_DEFAULT", "h7Qm!2Vx9#Lp4@Td")
	path := writeMarkdownFile(t, `---
title: Secret
private: true
---
# Secret
body
`)

	result, err := encryptPostSourceFile(path, cfg, false)
	if err != nil {
		t.Fatalf("encryptPostSourceFile() error = %v", err)
	}
	if result.Action != encryptPostActionEncrypted {
		t.Fatalf("action = %q, want %q", result.Action, encryptPostActionEncrypted)
	}

	content := readFileString(t, path)
	_, body, err := plugins.ExtractFrontmatter(content)
	if err != nil {
		t.Fatalf("ExtractFrontmatter() error = %v", err)
	}
	if !encryption.IsSourceEncrypted(body) {
		t.Fatalf("expected source-encrypted body, got %q", body)
	}
	decrypted, keyName, err := encryption.DecryptSourceMarkdown(body, "h7Qm!2Vx9#Lp4@Td")
	if err != nil {
		t.Fatalf("DecryptSourceMarkdown() error = %v", err)
	}
	if keyName != "default" {
		t.Fatalf("keyName = %q, want default", keyName)
	}
	if decrypted != "# Secret\nbody\n" {
		t.Fatalf("decrypted body = %q", decrypted)
	}
}

func TestEncryptPostSourceFile_DryRunDoesNotWrite(t *testing.T) {
	cfg := testEncryptPostsConfig()
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_DEFAULT", "h7Qm!2Vx9#Lp4@Td")
	original := `---
title: Secret
private: true
---
secret body
`
	path := writeMarkdownFile(t, original)

	result, err := encryptPostSourceFile(path, cfg, true)
	if err != nil {
		t.Fatalf("encryptPostSourceFile() error = %v", err)
	}
	if result.Action != encryptPostActionEncrypted {
		t.Fatalf("action = %q, want %q", result.Action, encryptPostActionEncrypted)
	}
	if got := readFileString(t, path); got != original {
		t.Fatalf("dry run modified file: got %q", got)
	}
}

func TestEncryptPostSourceFile_PrivateTagUsesTagKey(t *testing.T) {
	cfg := testEncryptPostsConfig()
	cfg.Encryption.PrivateTags = map[string]string{"diary": "personal"}
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_PERSONAL", "h7Qm!2Vx9#Lp4@Td")
	path := writeMarkdownFile(t, `---
title: Diary
tags:
  - diary
---
tagged secret
`)

	result, err := encryptPostSourceFile(path, cfg, false)
	if err != nil {
		t.Fatalf("encryptPostSourceFile() error = %v", err)
	}
	if result.KeyName != "personal" {
		t.Fatalf("key = %q, want personal", result.KeyName)
	}

	_, body, err := plugins.ExtractFrontmatter(readFileString(t, path))
	if err != nil {
		t.Fatalf("ExtractFrontmatter() error = %v", err)
	}
	_, keyName, err := encryption.DecryptSourceMarkdown(body, "h7Qm!2Vx9#Lp4@Td")
	if err != nil {
		t.Fatalf("DecryptSourceMarkdown() error = %v", err)
	}
	if keyName != "personal" {
		t.Fatalf("encrypted marker key = %q, want personal", keyName)
	}
}

func TestEncryptPostSourceFile_AlreadyEncryptedSkipsWithoutKey(t *testing.T) {
	cfg := testEncryptPostsConfig()
	encryptedBody, err := encryption.EncryptSourceMarkdown("secret body\n", "default", "h7Qm!2Vx9#Lp4@Td")
	if err != nil {
		t.Fatalf("EncryptSourceMarkdown() error = %v", err)
	}
	path := writeMarkdownFile(t, "---\ntitle: Secret\nprivate: true\n---\n"+encryptedBody)

	result, err := encryptPostSourceFile(path, cfg, false)
	if err != nil {
		t.Fatalf("encryptPostSourceFile() error = %v", err)
	}
	if result.Action != encryptPostActionAlreadyEncrypted {
		t.Fatalf("action = %q, want %q", result.Action, encryptPostActionAlreadyEncrypted)
	}
}

func TestEncryptPostSourceFile_WeakExplicitKeyFails(t *testing.T) {
	cfg := testEncryptPostsConfig()
	t.Setenv("MARKATA_GO_ENCRYPTION_KEY_PERSONAL", "weak")
	path := writeMarkdownFile(t, `---
title: Secret
private: true
secret_key: personal
---
secret body
`)

	_, err := encryptPostSourceFile(path, cfg, true)
	if err == nil {
		t.Fatal("expected weak key error")
	}
	if !strings.Contains(err.Error(), "failed policy") {
		t.Fatalf("expected policy error, got %v", err)
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

func testEncryptPostsConfig() *models.Config {
	return &models.Config{
		GlobConfig: models.GlobConfig{
			Patterns:     []string{"**/*.md"},
			UseGitignore: true,
		},
		Encryption: models.EncryptionConfig{
			Enabled:               true,
			DefaultKey:            "default",
			EnforceStrength:       true,
			MinPasswordLength:     14,
			MinEstimatedCrackTime: "10y",
		},
	}
}

func writeMarkdownFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "post.md")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	return path
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
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
		{name: "subsecond", in: 250 * time.Millisecond, want: "<1s"},
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
