package encryption

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestDeriveKey(t *testing.T) {
	tests := []struct {
		name     string
		password string
		salt     []byte
		wantErr  error
	}{
		{
			name:     "valid password and salt",
			password: "test-password-123",
			salt:     make([]byte, SaltSize),
			wantErr:  nil,
		},
		{
			name:     "empty password",
			password: "",
			salt:     make([]byte, SaltSize),
			wantErr:  ErrEmptyPassword,
		},
		{
			name:     "empty salt",
			password: "test-password",
			salt:     []byte{},
			wantErr:  ErrEmptySalt,
		},
		{
			name:     "invalid salt size",
			password: "test-password",
			salt:     make([]byte, 8), // Wrong size
			wantErr:  ErrInvalidSaltSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := DeriveKey(tt.password, tt.salt)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("DeriveKey() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("DeriveKey() unexpected error: %v", err)
			}

			if len(key) != KeySize {
				t.Errorf("DeriveKey() key size = %d, want %d", len(key), KeySize)
			}
		})
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	password := "my-secret-password"
	salt := []byte("0123456789abcdef") // 16 bytes

	key1, err := DeriveKey(password, salt)
	if err != nil {
		t.Fatalf("First DeriveKey() failed: %v", err)
	}

	key2, err := DeriveKey(password, salt)
	if err != nil {
		t.Fatalf("Second DeriveKey() failed: %v", err)
	}

	if !bytes.Equal(key1, key2) {
		t.Error("DeriveKey() should produce deterministic output for same inputs")
	}
}

func TestDeriveKey_DifferentSalts(t *testing.T) {
	password := "my-secret-password"
	salt1 := []byte("0123456789abcdef")
	salt2 := []byte("fedcba9876543210")

	key1, err := DeriveKey(password, salt1)
	if err != nil {
		t.Fatalf("First DeriveKey() failed: %v", err)
	}

	key2, err := DeriveKey(password, salt2)
	if err != nil {
		t.Fatalf("Second DeriveKey() failed: %v", err)
	}

	if bytes.Equal(key1, key2) {
		t.Error("DeriveKey() should produce different keys for different salts")
	}
}

func TestGenerateSalt(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error: %v", err)
	}

	if len(salt) != SaltSize {
		t.Errorf("GenerateSalt() size = %d, want %d", len(salt), SaltSize)
	}

	// Test randomness - generate another salt and ensure they're different
	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("Second GenerateSalt() error: %v", err)
	}

	if bytes.Equal(salt, salt2) {
		t.Error("GenerateSalt() should produce different salts on each call")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
		password  string
	}{
		{
			name:      "simple text",
			plaintext: "Hello, World!",
			password:  "test-password-123",
		},
		{
			name:      "unicode text",
			plaintext: "Hello, 世界! 🌍 Привет мир",
			password:  "unicode-password-测试",
		},
		{
			name:      "long text",
			plaintext: strings.Repeat("This is a longer test message. ", 100),
			password:  "another-password",
		},
		{
			name:      "markdown content",
			plaintext: "# Private Post\n\nThis is **confidential** content.\n\n- Item 1\n- Item 2",
			password:  "markdown-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := Encrypt([]byte(tt.plaintext), tt.password)
			if err != nil {
				t.Fatalf("Encrypt() error: %v", err)
			}

			// Verify it's base64 encoded
			_, err = base64.StdEncoding.DecodeString(ciphertext)
			if err != nil {
				t.Fatalf("Encrypt() output is not valid base64: %v", err)
			}

			// Decrypt
			decrypted, err := Decrypt(ciphertext, tt.password)
			if err != nil {
				t.Fatalf("Decrypt() error: %v", err)
			}

			// Verify roundtrip
			if string(decrypted) != tt.plaintext {
				t.Errorf("Roundtrip failed:\ngot:  %q\nwant: %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncrypt_Randomness(t *testing.T) {
	plaintext := []byte("Same message encrypted twice")
	password := "test-password"

	cipher1, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("First Encrypt() error: %v", err)
	}

	cipher2, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Second Encrypt() error: %v", err)
	}

	// Should be different due to random salt and nonce
	if cipher1 == cipher2 {
		t.Error("Encrypt() should produce different ciphertexts for same plaintext (random salt/nonce)")
	}

	// Both should decrypt to the same plaintext
	decrypted1, err := Decrypt(cipher1, password)
	if err != nil {
		t.Fatalf("Decrypt() cipher1 error: %v", err)
	}

	decrypted2, err := Decrypt(cipher2, password)
	if err != nil {
		t.Fatalf("Decrypt() cipher2 error: %v", err)
	}

	if !bytes.Equal(decrypted1, decrypted2) {
		t.Error("Different ciphertexts should decrypt to same plaintext")
	}
}

func TestEncrypt_Errors(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
		password  string
		wantErr   error
	}{
		{
			name:      "empty plaintext",
			plaintext: []byte{},
			password:  "password",
			wantErr:   ErrEmptyPlaintext,
		},
		{
			name:      "nil plaintext",
			plaintext: nil,
			password:  "password",
			wantErr:   ErrEmptyPlaintext,
		},
		{
			name:      "empty password",
			plaintext: []byte("test"),
			password:  "",
			wantErr:   ErrEmptyPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Encrypt(tt.plaintext, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecrypt_Errors(t *testing.T) {
	tests := []struct {
		name       string
		ciphertext string
		password   string
		wantErr    error
	}{
		{
			name:       "empty ciphertext",
			ciphertext: "",
			password:   "password",
			wantErr:    ErrEmptyCiphertext,
		},
		{
			name:       "empty password",
			ciphertext: "dGVzdA==",
			password:   "",
			wantErr:    ErrEmptyPassword,
		},
		{
			name:       "invalid base64",
			ciphertext: "not-valid-base64!!!",
			password:   "password",
			wantErr:    nil, // Will be wrapped error
		},
		{
			name:       "too short ciphertext",
			ciphertext: base64.StdEncoding.EncodeToString([]byte("short")),
			password:   "password",
			wantErr:    ErrMalformedData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.ciphertext, tt.password)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if err == nil {
				t.Error("Decrypt() expected an error, got nil")
			}
		})
	}
}

func TestDecrypt_WrongPassword(t *testing.T) {
	plaintext := []byte("Secret message")
	correctPassword := "correct-password"
	wrongPassword := "wrong-password"

	ciphertext, err := Encrypt(plaintext, correctPassword)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	_, err = Decrypt(ciphertext, wrongPassword)
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("Decrypt() with wrong password error = %v, want %v", err, ErrDecryptionFailed)
	}
}

func TestDecrypt_TamperedData(t *testing.T) {
	plaintext := []byte("Secret message")
	password := "test-password"

	ciphertext, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	// Tamper with the ciphertext
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decode ciphertext: %v", err)
	}
	decoded[len(decoded)-1] ^= 0xFF // Flip bits in the last byte (auth tag)
	tampered := base64.StdEncoding.EncodeToString(decoded)

	_, err = Decrypt(tampered, password)
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("Decrypt() with tampered data error = %v, want %v", err, ErrDecryptionFailed)
	}
}

func TestEncryptDecryptWithKey(t *testing.T) {
	plaintext := []byte("Test message")
	password := "test-password"

	// Generate salt and derive key
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error: %v", err)
	}

	key, err := DeriveKey(password, salt)
	if err != nil {
		t.Fatalf("DeriveKey() error: %v", err)
	}

	// Encrypt with key
	ciphertext, err := EncryptWithKey(plaintext, key, salt)
	if err != nil {
		t.Fatalf("EncryptWithKey() error: %v", err)
	}

	// Decrypt with key
	decrypted, err := DecryptWithKey(ciphertext, key)
	if err != nil {
		t.Fatalf("DecryptWithKey() error: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptWithKey_Errors(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error: %v", err)
	}

	tests := []struct {
		name      string
		plaintext []byte
		key       []byte
		salt      []byte
		wantErr   error
	}{
		{
			name:      "empty plaintext",
			plaintext: []byte{},
			key:       make([]byte, KeySize),
			salt:      salt,
			wantErr:   ErrEmptyPlaintext,
		},
		{
			name:      "invalid key size",
			plaintext: []byte("test"),
			key:       make([]byte, 16), // Wrong size
			salt:      salt,
			wantErr:   ErrInvalidKey,
		},
		{
			name:      "invalid salt size",
			plaintext: []byte("test"),
			key:       make([]byte, KeySize),
			salt:      make([]byte, 8),
			wantErr:   ErrInvalidSaltSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncryptWithKey(tt.plaintext, tt.key, tt.salt)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("EncryptWithKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecryptWithKey_Errors(t *testing.T) {
	tests := []struct {
		name       string
		ciphertext string
		key        []byte
		wantErr    error
	}{
		{
			name:       "empty ciphertext",
			ciphertext: "",
			key:        make([]byte, KeySize),
			wantErr:    ErrEmptyCiphertext,
		},
		{
			name:       "invalid key size",
			ciphertext: base64.StdEncoding.EncodeToString(make([]byte, 100)),
			key:        make([]byte, 16),
			wantErr:    ErrInvalidKey,
		},
		{
			name:       "too short ciphertext",
			ciphertext: base64.StdEncoding.EncodeToString([]byte("short")),
			key:        make([]byte, KeySize),
			wantErr:    ErrMalformedData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecryptWithKey(tt.ciphertext, tt.key)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("DecryptWithKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Benchmark encryption/decryption performance
func BenchmarkEncrypt(b *testing.B) {
	plaintext := []byte(strings.Repeat("This is a test message for benchmarking. ", 100))
	password := "benchmark-password"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Encrypt(plaintext, password) //nolint:errcheck // benchmark
	}
}

func BenchmarkDecrypt(b *testing.B) {
	plaintext := []byte(strings.Repeat("This is a test message for benchmarking. ", 100))
	password := "benchmark-password"

	ciphertext, _ := Encrypt(plaintext, password) //nolint:errcheck // setup

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Decrypt(ciphertext, password) //nolint:errcheck // benchmark
	}
}

func BenchmarkDeriveKey(b *testing.B) {
	password := "benchmark-password"
	salt := make([]byte, SaltSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DeriveKey(password, salt) //nolint:errcheck // benchmark
	}
}
