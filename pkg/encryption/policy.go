package encryption

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	// DefaultMinEstimatedCrackTime is the default threshold for estimated crack time.
	DefaultMinEstimatedCrackTime = "10y"

	// DefaultMinPasswordLength is the default minimum password length.
	DefaultMinPasswordLength = 14

	// defaultGuessRate is the assumed attacker guess rate (guesses per second).
	defaultGuessRate = 10_000_000_000

	// symbolCharsetSize approximates the number of printable symbols used in passwords.
	symbolCharsetSize = 32
)

var (
	// DefaultMinEstimatedCrackDuration matches DefaultMinEstimatedCrackTime (365 days/year).
	DefaultMinEstimatedCrackDuration = 10 * 365 * 24 * time.Hour

	durationTokenRe  = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)([smhdy])`)
	durationSuffixes = map[string]float64{
		"s": 1,
		"m": 60,
		"h": 3600,
		"d": 24 * 3600,
		"y": 365 * 24 * 3600,
	}
	defaultGuessRateLog2 = math.Log2(defaultGuessRate)
)

// EstimateCrackTime returns the estimated duration it takes to brute-force the password
// assuming defaultGuessRate guesses per second.
func EstimateCrackTime(password string) time.Duration {
	if password == "" {
		return 0
	}

	charset := charsetSize(password)
	if charset <= 0 {
		return 0
	}

	entropy := float64(len(password)) * math.Log2(charset)
	seconds := math.Exp2(entropy - defaultGuessRateLog2)
	if math.IsInf(seconds, 0) || math.IsNaN(seconds) {
		return time.Duration(math.MaxInt64)
	}

	nanos := seconds * float64(time.Second)
	if nanos <= 0 {
		return 0
	}
	if nanos > float64(math.MaxInt64) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(nanos)
}

// ValidatePassword checks the password against length and crack-time thresholds.
func ValidatePassword(password string, minLength int, minDuration time.Duration) error {
	if minLength > 0 && len(password) < minLength {
		return fmt.Errorf("password length %d < required %d", len(password), minLength)
	}
	if minDuration > 0 {
		estimated := EstimateCrackTime(password)
		if estimated < minDuration {
			return fmt.Errorf("estimated crack time %s < required %s", formatDuration(estimated), formatDuration(minDuration))
		}
	}
	return nil
}

// ParseEstimatedCrackDuration parses durations that support y/d/h/m/s units.
func ParseEstimatedCrackDuration(value string) (time.Duration, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("invalid duration: empty")
	}

	if d, err := time.ParseDuration(trimmed); err == nil {
		return d, nil
	}

	normalized := strings.ReplaceAll(trimmed, " ", "")
	matches := durationTokenRe.FindAllStringSubmatch(normalized, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid duration: %s", value)
	}

	totalChars := 0
	totalSeconds := 0.0
	for _, match := range matches {
		totalChars += len(match[0])
		amount, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration value %q", match[1])
		}
		unit := strings.ToLower(match[2])
		multiplier, ok := durationSuffixes[unit]
		if !ok {
			return 0, fmt.Errorf("invalid duration unit %q", unit)
		}
		totalSeconds += amount * multiplier
	}

	if totalChars != len(normalized) {
		return 0, fmt.Errorf("invalid duration: %s", value)
	}

	nanos := totalSeconds * float64(time.Second)
	if nanos > float64(math.MaxInt64) {
		return time.Duration(math.MaxInt64), nil
	}
	return time.Duration(nanos), nil
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	return d.Round(time.Second).String()
}

func charsetSize(password string) float64 {
	var hasLower, hasUpper, hasDigit, hasSymbol bool
	unique := make(map[rune]struct{})
	for _, r := range password {
		unique[r] = struct{}{}
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			hasSymbol = true
		}
	}
	pool := 0
	if hasLower {
		pool += 26
	}
	if hasUpper {
		pool += 26
	}
	if hasDigit {
		pool += 10
	}
	if hasSymbol {
		pool += symbolCharsetSize
	}
	if pool == 0 {
		pool = len(unique)
	}
	if len(unique) > pool {
		pool = len(unique)
	}
	if pool == 0 {
		pool = 1
	}
	return float64(pool)
}
