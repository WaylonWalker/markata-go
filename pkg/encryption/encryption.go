// Package encryption provides AES-256-GCM encryption utilities for private posts.
//
// # Overview
//
// This package implements symmetric encryption using AES-256-GCM (Galois/Counter Mode),
// which provides both confidentiality and authenticity. The encryption workflow is:
//
// 1. Derive a 256-bit key from a password using PBKDF2 with SHA-256
// 2. Generate a random 12-byte nonce for each encryption
// 3. Encrypt plaintext using AES-256-GCM
// 4. Output: base64(salt + nonce + ciphertext + tag)
//
// # Client-Side Decryption
//
// The encrypted content can be decrypted in the browser using the Web Crypto API.
// The salt, nonce, and ciphertext are concatenated and base64-encoded for easy
// transmission. The same PBKDF2 parameters must be used for key derivation.
//
// # Security Notes
//
// - Use a unique password/key for each sensitivity level
// - The encryption protects content, not metadata (title, description remain public)
// - Keys should be stored in environment variables, never committed to source control
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

// Constants for encryption parameters.
const (
	// SaltSize is the size of the salt in bytes.
	SaltSize = 16

	// NonceSize is the size of the GCM nonce in bytes.
	// GCM recommends a 12-byte nonce for optimal performance and security.
	NonceSize = 12

	// KeySize is the size of the AES-256 key in bytes.
	KeySize = 32

	// PBKDF2Iterations is the number of PBKDF2 iterations for key derivation.
	// 100,000 iterations provides a good balance between security and performance.
	// This must match the client-side JavaScript implementation.
	PBKDF2Iterations = 100000
)

// Common errors for encryption operations.
var (
	ErrEmptyPassword    = errors.New("encryption: password cannot be empty")
	ErrEmptySalt        = errors.New("encryption: salt cannot be empty")
	ErrInvalidSaltSize  = errors.New("encryption: salt must be 16 bytes")
	ErrEmptyPlaintext   = errors.New("encryption: plaintext cannot be empty")
	ErrEmptyCiphertext  = errors.New("encryption: ciphertext cannot be empty")
	ErrInvalidKey       = errors.New("encryption: key must be 32 bytes")
	ErrMalformedData    = errors.New("encryption: ciphertext too short to contain salt, nonce, and tag")
	ErrDecryptionFailed = errors.New("encryption: decryption failed (wrong password or corrupted data)")
)

// DeriveKey derives a 256-bit encryption key from a password and salt using PBKDF2.
// The salt should be a random 16-byte value. For encryption, generate a new salt.
// For decryption, extract the salt from the ciphertext header.
func DeriveKey(password string, salt []byte) ([]byte, error) {
	if password == "" {
		return nil, ErrEmptyPassword
	}
	if len(salt) == 0 {
		return nil, ErrEmptySalt
	}
	if len(salt) != SaltSize {
		return nil, ErrInvalidSaltSize
	}

	key := pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeySize, sha256.New)
	return key, nil
}

// GenerateSalt generates a cryptographically secure random salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("encryption: failed to generate salt: %w", err)
	}
	return salt, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded string.
// The output format is: base64(salt || nonce || ciphertext || tag)
// where || denotes concatenation.
//
// The key must be a 32-byte AES-256 key (use DeriveKey to create from password).
// A new random salt and nonce are generated for each call.
func Encrypt(plaintext []byte, password string) (string, error) {
	if len(plaintext) == 0 {
		return "", ErrEmptyPlaintext
	}
	if password == "" {
		return "", ErrEmptyPassword
	}

	// Generate random salt
	salt, err := GenerateSalt()
	if err != nil {
		return "", err
	}

	// Derive key from password
	key, err := DeriveKey(password, salt)
	if err != nil {
		return "", err
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("encryption: failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("encryption: failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("encryption: failed to generate nonce: %w", err)
	}

	// Encrypt (GCM appends the authentication tag to the ciphertext)
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Combine: salt || nonce || ciphertext (which includes tag)
	combined := make([]byte, SaltSize+NonceSize+len(ciphertext))
	copy(combined[0:SaltSize], salt)
	copy(combined[SaltSize:SaltSize+NonceSize], nonce)
	copy(combined[SaltSize+NonceSize:], ciphertext)

	// Encode as base64
	return base64.StdEncoding.EncodeToString(combined), nil
}

// EncryptWithKey encrypts plaintext using a pre-derived key.
// The output format is: base64(salt || nonce || ciphertext || tag)
// where the salt is provided for storage with the ciphertext.
//
// The key must be a 32-byte AES-256 key.
func EncryptWithKey(plaintext, key, salt []byte) (string, error) {
	if len(plaintext) == 0 {
		return "", ErrEmptyPlaintext
	}
	if len(key) != KeySize {
		return "", ErrInvalidKey
	}
	if len(salt) != SaltSize {
		return "", ErrInvalidSaltSize
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("encryption: failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("encryption: failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("encryption: failed to generate nonce: %w", err)
	}

	// Encrypt (GCM appends the authentication tag to the ciphertext)
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Combine: salt || nonce || ciphertext (which includes tag)
	combined := make([]byte, SaltSize+NonceSize+len(ciphertext))
	copy(combined[0:SaltSize], salt)
	copy(combined[SaltSize:SaltSize+NonceSize], nonce)
	copy(combined[SaltSize+NonceSize:], ciphertext)

	// Encode as base64
	return base64.StdEncoding.EncodeToString(combined), nil
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM.
// The input format must be: base64(salt || nonce || ciphertext || tag)
//
// The password is used with PBKDF2 to derive the decryption key.
func Decrypt(ciphertext64, password string) ([]byte, error) {
	if ciphertext64 == "" {
		return nil, ErrEmptyCiphertext
	}
	if password == "" {
		return nil, ErrEmptyPassword
	}

	// Decode base64
	combined, err := base64.StdEncoding.DecodeString(ciphertext64)
	if err != nil {
		return nil, fmt.Errorf("encryption: invalid base64: %w", err)
	}

	// Minimum size: salt + nonce + tag (16 bytes minimum for GCM tag)
	minSize := SaltSize + NonceSize + 16
	if len(combined) < minSize {
		return nil, ErrMalformedData
	}

	// Extract salt, nonce, and ciphertext
	salt := combined[0:SaltSize]
	nonce := combined[SaltSize : SaltSize+NonceSize]
	ciphertext := combined[SaltSize+NonceSize:]

	// Derive key from password
	key, err := DeriveKey(password, salt)
	if err != nil {
		return nil, err
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("encryption: failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encryption: failed to create GCM: %w", err)
	}

	// Decrypt (GCM verifies the authentication tag)
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// DecryptWithKey decrypts base64-encoded ciphertext using a pre-derived key.
// The input format must be: base64(salt || nonce || ciphertext || tag)
//
// The key must be a 32-byte AES-256 key.
func DecryptWithKey(ciphertext64 string, key []byte) ([]byte, error) {
	if ciphertext64 == "" {
		return nil, ErrEmptyCiphertext
	}
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	// Decode base64
	combined, err := base64.StdEncoding.DecodeString(ciphertext64)
	if err != nil {
		return nil, fmt.Errorf("encryption: invalid base64: %w", err)
	}

	// Minimum size: salt + nonce + tag (16 bytes minimum for GCM tag)
	minSize := SaltSize + NonceSize + 16
	if len(combined) < minSize {
		return nil, ErrMalformedData
	}

	// Extract nonce and ciphertext (skip salt, key already derived)
	nonce := combined[SaltSize : SaltSize+NonceSize]
	ciphertext := combined[SaltSize+NonceSize:]

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("encryption: failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encryption: failed to create GCM: %w", err)
	}

	// Decrypt (GCM verifies the authentication tag)
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}
