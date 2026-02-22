package encryption

import (
	"strings"
	"testing"
	"time"
)

func TestParseEstimatedCrackDuration(t *testing.T) {
	cases := []struct {
		input string
		want  time.Duration
	}{
		{"1y", 365 * 24 * time.Hour},
		{"1y6d", (365 + 6) * 24 * time.Hour},
		{"2h30m", 2*time.Hour + 30*time.Minute},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			dur, err := ParseEstimatedCrackDuration(tc.input)
			if err != nil {
				t.Fatalf("ParseEstimatedCrackDuration(%q) error = %v", tc.input, err)
			}
			if dur != tc.want {
				t.Errorf("duration = %v, want %v", dur, tc.want)
			}
		})
	}

	if _, err := ParseEstimatedCrackDuration("invalid"); err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestValidatePasswordLength(t *testing.T) {
	short := strings.Repeat("a", DefaultMinPasswordLength-1)
	if err := ValidatePassword(short, DefaultMinPasswordLength, DefaultMinEstimatedCrackDuration); err == nil {
		t.Error("expected length validation error")
	}
}

func TestValidatePasswordCrackTime(t *testing.T) {
	weak := "Password123!" // low entropy relative to default duration
	if err := ValidatePassword(weak, DefaultMinPasswordLength, DefaultMinEstimatedCrackDuration); err == nil {
		t.Error("expected crack time validation error")
	}
}

func TestGeneratePassword(t *testing.T) {
	password, err := GeneratePassword(DefaultMinPasswordLength)
	if err != nil {
		t.Fatalf("GeneratePassword() error = %v", err)
	}
	if len(password) != DefaultMinPasswordLength {
		t.Errorf("password length = %d, want %d", len(password), DefaultMinPasswordLength)
	}
	if err := ValidatePassword(password, DefaultMinPasswordLength, DefaultMinEstimatedCrackDuration); err != nil {
		t.Fatalf("generated password failed validation: %v", err)
	}
}
