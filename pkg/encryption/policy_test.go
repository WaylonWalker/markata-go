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
	weak := "password12345"
	if err := ValidatePassword(weak, 0, 10*365*24*time.Hour); err == nil {
		t.Error("expected crack time validation error")
	}
}

func TestEstimateCrackTime_CommonPasswordIsFast(t *testing.T) {
	estimated := EstimateCrackTime("password12345")
	if estimated <= 0 {
		t.Fatalf("EstimateCrackTime returned %v", estimated)
	}
	if estimated > 365*24*time.Hour {
		t.Fatalf("expected common password to crack in under a year, got %v", estimated)
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
