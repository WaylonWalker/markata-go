package encryption

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	lowerChars  = "abcdefghijklmnopqrstuvwxyz"
	upperChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars  = "0123456789"
	symbolChars = "!@#$%^&*()-_=+[]{}<>?,./"
)

var allChars = strings.Join([]string{lowerChars, upperChars, digitChars, symbolChars}, "")

// GeneratePassword returns a random password that satisfies the default policy.
func GeneratePassword(length int) (string, error) {
	if length < DefaultMinPasswordLength {
		return "", fmt.Errorf("length must be at least %d", DefaultMinPasswordLength)
	}

	categories := []string{lowerChars, upperChars, digitChars, symbolChars}
	maxAttempts := 5
	for attempt := 0; attempt < maxAttempts; attempt++ {
		password := make([]byte, length)
		idx := 0
		for _, cat := range categories {
			ch, err := randomCharFrom(cat)
			if err != nil {
				return "", err
			}
			password[idx] = ch
			idx++
		}
		for ; idx < length; idx++ {
			ch, err := randomCharFrom(allChars)
			if err != nil {
				return "", err
			}
			password[idx] = ch
		}
		if err := shuffle(password); err != nil {
			return "", err
		}
		candidate := string(password)
		if err := ValidatePassword(candidate, DefaultMinPasswordLength, DefaultMinEstimatedCrackDuration); err != nil {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("failed to generate a compliant password")
}

func randomCharFrom(chars string) (byte, error) {
	idx, err := randomInt(len(chars))
	if err != nil {
		return 0, err
	}
	return chars[idx], nil
}

func randomInt(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("invalid range %d", n)
	}
	upperBound := big.NewInt(int64(n))
	num, err := rand.Int(rand.Reader, upperBound)
	if err != nil {
		return 0, err
	}
	return int(num.Int64()), nil
}

func shuffle(data []byte) error {
	for i := len(data) - 1; i > 0; i-- {
		j, err := randomInt(i + 1)
		if err != nil {
			return err
		}
		data[i], data[j] = data[j], data[i]
	}
	return nil
}
