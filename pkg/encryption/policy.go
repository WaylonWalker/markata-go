package encryption

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	zxcvbn "github.com/nbutton23/zxcvbn-go"
)

const (
	// DefaultMinEstimatedCrackTime is the default threshold for estimated crack time.
	DefaultMinEstimatedCrackTime = "10y"

	// DefaultMinPasswordLength is the default minimum password length.
	DefaultMinPasswordLength = 14

	// zxcvbnSecondsPerGuess aligns with zxcvbn-go's offline slow-hash model:
	// 10ms/guess across 100 parallel attackers => 10,000 guesses/sec.
	zxcvbnSecondsPerGuess = 0.0001
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
)

// EstimateCrackTime returns the estimated duration it takes to brute-force the password
// assuming defaultGuessRate guesses per second.
func EstimateCrackTime(password string) time.Duration {
	if password == "" {
		return 0
	}

	result := zxcvbn.PasswordStrength(password, nil)
	seconds := result.CrackTime
	if seconds <= 0 && !math.IsNaN(result.Entropy) && result.Entropy > 0 {
		seconds = (0.5 * math.Exp2(result.Entropy)) * zxcvbnSecondsPerGuess
	}
	if !math.IsNaN(result.Entropy) && result.Entropy > 0 {
		entropyUpperBoundSeconds := (0.5 * math.Exp2(result.Entropy)) * zxcvbnSecondsPerGuess
		if entropyUpperBoundSeconds > 0 && entropyUpperBoundSeconds < seconds {
			seconds = entropyUpperBoundSeconds
		}
	}
	if seconds <= 0 {
		return 0
	}
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
