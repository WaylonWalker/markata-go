// Package palettes provides color palette management for markata-go.
package palettes

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Lospec-related errors.
var (
	// ErrInvalidLospecURL is returned when a URL is not a valid Lospec palette URL.
	ErrInvalidLospecURL = errors.New("invalid Lospec URL: must be https://lospec.com/palette-list/<name>.txt")

	// ErrLospecFetchFailed is returned when fetching from Lospec fails.
	ErrLospecFetchFailed = errors.New("failed to fetch from Lospec")

	// ErrLospecParseError is returned when the Lospec response cannot be parsed.
	ErrLospecParseError = errors.New("failed to parse Lospec palette")
)

// LospecFetchError provides context for Lospec fetch failures.
type LospecFetchError struct {
	URL        string
	StatusCode int
	Message    string
	Err        error
}

func (e *LospecFetchError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("lospec fetch failed for %s: HTTP %d: %s", e.URL, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("lospec fetch failed for %s: %s", e.URL, e.Message)
}

func (e *LospecFetchError) Unwrap() error {
	return e.Err
}

// NewLospecFetchError creates a new LospecFetchError.
func NewLospecFetchError(url string, statusCode int, message string, err error) *LospecFetchError {
	return &LospecFetchError{
		URL:        url,
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}

// lospecURLPattern matches valid Lospec palette URLs.
// Format: https://lospec.com/palette-list/<name>.txt
var lospecURLPattern = regexp.MustCompile(`^https://lospec\.com/palette-list/([a-zA-Z0-9_-]+)\.txt$`)

// LospecClient handles fetching palettes from Lospec.com.
type LospecClient struct {
	httpClient *http.Client
	cacheDir   string
	userAgent  string
}

// LospecClientOption is a functional option for configuring LospecClient.
type LospecClientOption func(*LospecClient)

// WithLospecTimeout sets the HTTP client timeout.
func WithLospecTimeout(timeout time.Duration) LospecClientOption {
	return func(c *LospecClient) {
		c.httpClient.Timeout = timeout
	}
}

// WithLospecCacheDir sets the cache directory for downloaded palettes.
func WithLospecCacheDir(dir string) LospecClientOption {
	return func(c *LospecClient) {
		c.cacheDir = dir
	}
}

// WithLospecUserAgent sets the User-Agent header for requests.
func WithLospecUserAgent(ua string) LospecClientOption {
	return func(c *LospecClient) {
		c.userAgent = ua
	}
}

// NewLospecClient creates a new LospecClient with default settings.
func NewLospecClient(opts ...LospecClientOption) *LospecClient {
	// Default cache directory: ~/.cache/markata-go/lospec/
	cacheDir := ""
	if userCacheDir, err := os.UserCacheDir(); err == nil {
		cacheDir = filepath.Join(userCacheDir, "markata-go", "lospec")
	}

	c := &LospecClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheDir:  cacheDir,
		userAgent: "markata-go/1.0",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ValidateLospecURL checks if a URL is a valid Lospec palette URL.
func ValidateLospecURL(rawURL string) error {
	if !lospecURLPattern.MatchString(rawURL) {
		return ErrInvalidLospecURL
	}
	return nil
}

// ExtractPaletteNameFromURL extracts the palette name from a Lospec URL.
func ExtractPaletteNameFromURL(rawURL string) (string, error) {
	matches := lospecURLPattern.FindStringSubmatch(rawURL)
	if len(matches) < 2 {
		return "", ErrInvalidLospecURL
	}
	// Convert kebab-case to title case for display
	name := matches[1]
	name = strings.ReplaceAll(name, "-", " ")
	// Capitalize first letter of each word
	words := strings.Fields(name)
	for i, word := range words {
		if word != "" {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " "), nil
}

// FetchPalette fetches a palette from a Lospec URL.
// It uses caching to avoid repeated requests for the same palette.
func (c *LospecClient) FetchPalette(ctx context.Context, rawURL string) (*Palette, error) {
	// Validate URL
	if err := ValidateLospecURL(rawURL); err != nil {
		return nil, err
	}

	// Check cache first
	if c.cacheDir != "" {
		if cached, err := c.loadFromCache(rawURL); err == nil {
			return cached, nil
		}
	}

	// Fetch from Lospec
	colors, err := c.fetchColors(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	// Extract palette name from URL
	name, err := ExtractPaletteNameFromURL(rawURL)
	if err != nil {
		return nil, err
	}

	// Create palette with auto-generated semantic mappings
	palette := CreatePaletteFromColors(name, colors, rawURL)

	// Cache the result
	if c.cacheDir != "" {
		_ = c.saveToCache(rawURL, colors) //nolint:errcheck // Best effort caching
	}

	return palette, nil
}

// fetchColors fetches the raw color list from Lospec.
func (c *LospecClient) fetchColors(ctx context.Context, rawURL string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return nil, NewLospecFetchError(rawURL, 0, "failed to create request", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, NewLospecFetchError(rawURL, 0, "request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewLospecFetchError(rawURL, resp.StatusCode, "unexpected status", ErrLospecFetchFailed)
	}

	return parseLospecColors(resp.Body)
}

// parseLospecColors parses the Lospec text format.
// Lospec returns one hex color per line, optionally with # prefix.
// Colors may be in ARGB format (8 characters) or RGB format (6 characters).
// Lines starting with ; are comments and are skipped.
func parseLospecColors(r io.Reader) ([]string, error) {
	var colors []string
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Skip comment lines (Paint.NET palette format)
		if strings.HasPrefix(line, ";") {
			continue
		}

		// Normalize: remove # prefix if present
		color := strings.TrimPrefix(line, "#")

		// Handle ARGB format (8 chars) - strip the alpha channel
		if len(color) == 8 {
			color = color[2:] // Remove first 2 chars (alpha)
		}

		// Ensure # prefix for validation
		color = "#" + color

		// Validate hex color
		if !isHexColor(color) {
			continue // Skip invalid lines
		}

		colors = append(colors, normalizeHexColor(color))
	}

	if err := scanner.Err(); err != nil {
		return nil, NewLospecFetchError("", 0, "failed to read response", err)
	}

	if len(colors) == 0 {
		return nil, NewLospecFetchError("", 0, "no valid colors found", ErrLospecParseError)
	}

	return colors, nil
}

// cacheKey generates a cache key from a URL.
func cacheKey(rawURL string) string {
	h := sha256.Sum256([]byte(rawURL))
	return hex.EncodeToString(h[:8]) // Use first 8 bytes for shorter filename
}

// loadFromCache attempts to load colors from the cache.
func (c *LospecClient) loadFromCache(rawURL string) (*Palette, error) {
	cacheFile := filepath.Join(c.cacheDir, cacheKey(rawURL)+".txt")

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	colors, err := parseLospecColors(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	name, err := ExtractPaletteNameFromURL(rawURL)
	if err != nil {
		return nil, err
	}

	return CreatePaletteFromColors(name, colors, rawURL), nil
}

// saveToCache saves colors to the cache.
func (c *LospecClient) saveToCache(rawURL string, colors []string) error {
	if err := os.MkdirAll(c.cacheDir, 0o755); err != nil {
		return err
	}

	cacheFile := filepath.Join(c.cacheDir, cacheKey(rawURL)+".txt")
	content := strings.Join(colors, "\n") + "\n"

	return os.WriteFile(cacheFile, []byte(content), 0o644) //nolint:gosec // cache files should be readable
}

// CreatePaletteFromColors creates a Palette from a list of hex colors.
// It auto-generates semantic mappings based on color analysis.
func CreatePaletteFromColors(name string, colors []string, sourceURL string) *Palette {
	p := NewPalette(name, VariantDark) // Default to dark, will be auto-detected
	p.Homepage = sourceURL
	p.Description = "Imported from Lospec"

	// Analyze colors to determine variant
	avgLuminance := 0.0
	for _, color := range colors {
		avgLuminance += relativeLuminance(color)
	}
	avgLuminance /= float64(len(colors))

	// If average luminance is high, it's likely a light theme
	if avgLuminance > 0.5 {
		p.Variant = VariantLight
	}

	// Sort colors by luminance for better semantic assignment
	type colorWithLum struct {
		hex string
		lum float64
	}
	sortedColors := make([]colorWithLum, len(colors))
	for i, color := range colors {
		sortedColors[i] = colorWithLum{
			hex: color,
			lum: relativeLuminance(color),
		}
	}
	sort.Slice(sortedColors, func(i, j int) bool {
		return sortedColors[i].lum < sortedColors[j].lum
	})

	// Assign raw colors with indexed names
	for i, color := range colors {
		colorName := fmt.Sprintf("color%d", i)
		p.Colors[colorName] = color
	}

	// Generate semantic mappings based on luminance
	numColors := len(sortedColors)
	if numColors == 0 {
		return p
	}

	// For dark themes: darkest = bg, lightest = text
	// For light themes: lightest = bg, darkest = text
	if p.Variant == VariantDark {
		// Background: darkest color
		p.Semantic["bg-primary"] = findColorName(p.Colors, sortedColors[0].hex)
		// Text: lightest color
		p.Semantic["text-primary"] = findColorName(p.Colors, sortedColors[numColors-1].hex)
	} else {
		// Background: lightest color
		p.Semantic["bg-primary"] = findColorName(p.Colors, sortedColors[numColors-1].hex)
		// Text: darkest color
		p.Semantic["text-primary"] = findColorName(p.Colors, sortedColors[0].hex)
	}

	// Accent: pick from middle colors, preferring more saturated ones
	if numColors >= 3 {
		// Find most saturated color in the middle range
		midStart := numColors / 4
		midEnd := numColors - numColors/4
		if midStart == midEnd {
			midEnd = midStart + 1
		}

		bestAccent := sortedColors[numColors/2].hex
		bestSaturation := 0.0

		for i := midStart; i < midEnd && i < numColors; i++ {
			sat := saturation(sortedColors[i].hex)
			if sat > bestSaturation {
				bestSaturation = sat
				bestAccent = sortedColors[i].hex
			}
		}
		p.Semantic["accent"] = findColorName(p.Colors, bestAccent)
	} else if numColors >= 2 {
		// With only 2 colors, use the one that's not bg or text
		for _, sc := range sortedColors {
			name := findColorName(p.Colors, sc.hex)
			if name != p.Semantic["bg-primary"] && name != p.Semantic["text-primary"] {
				p.Semantic["accent"] = name
				break
			}
		}
	}

	// Link color: accent or a distinct color
	if accent, ok := p.Semantic["accent"]; ok {
		p.Semantic["link"] = accent
	}

	// Secondary colors if we have enough
	if numColors >= 4 {
		// Secondary background: slightly different from primary
		if p.Variant == VariantDark {
			p.Semantic["bg-secondary"] = findColorName(p.Colors, sortedColors[1].hex)
		} else {
			p.Semantic["bg-secondary"] = findColorName(p.Colors, sortedColors[numColors-2].hex)
		}

		// Secondary text: slightly different from primary
		if p.Variant == VariantDark {
			p.Semantic["text-secondary"] = findColorName(p.Colors, sortedColors[numColors-2].hex)
		} else {
			p.Semantic["text-secondary"] = findColorName(p.Colors, sortedColors[1].hex)
		}
	}

	return p
}

// findColorName finds the color name in the map that matches the hex value.
func findColorName(colors map[string]string, hex string) string {
	for name, value := range colors {
		if value == hex {
			return name
		}
	}
	return ""
}

// saturation calculates a simple saturation metric for a hex color.
// Returns a value between 0 (grayscale) and 1 (fully saturated).
func saturation(hex string) float64 {
	r, g, b := hexToRGB(hex)
	maxC := max(r, max(g, b))
	minC := min(r, min(g, b))

	if maxC == 0 {
		return 0
	}

	return float64(maxC-minC) / float64(maxC)
}

// hexToRGB converts a hex color to RGB components (0-255).
func hexToRGB(hex string) (r, g, b uint8) {
	hex = strings.TrimPrefix(hex, "#")

	// Handle short form (#RGB)
	if len(hex) == 3 {
		hex = string(hex[0]) + string(hex[0]) +
			string(hex[1]) + string(hex[1]) +
			string(hex[2]) + string(hex[2])
	}

	if len(hex) < 6 {
		return 0, 0, 0
	}

	var ri, gi, bi int64
	_, _ = fmt.Sscanf(hex[0:2], "%x", &ri) //nolint:errcheck // hex already validated
	_, _ = fmt.Sscanf(hex[2:4], "%x", &gi) //nolint:errcheck // hex already validated
	_, _ = fmt.Sscanf(hex[4:6], "%x", &bi) //nolint:errcheck // hex already validated

	return uint8(ri), uint8(gi), uint8(bi) //nolint:gosec // values are bounded by hex parsing
}

// relativeLuminance calculates the relative luminance of a color.
// Based on WCAG 2.1 definition.
func relativeLuminance(hex string) float64 {
	r, g, b := hexToRGB(hex)

	// Convert to sRGB
	sR := float64(r) / 255.0
	sG := float64(g) / 255.0
	sB := float64(b) / 255.0

	// Apply gamma correction
	if sR <= 0.03928 {
		sR /= 12.92
	} else {
		sR = pow((sR+0.055)/1.055, 2.4)
	}
	if sG <= 0.03928 {
		sG /= 12.92
	} else {
		sG = pow((sG+0.055)/1.055, 2.4)
	}
	if sB <= 0.03928 {
		sB /= 12.92
	} else {
		sB = pow((sB+0.055)/1.055, 2.4)
	}

	return 0.2126*sR + 0.7152*sG + 0.0722*sB
}

// pow is a simple power function to avoid math import for this small usage.
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	// Handle fractional exponent approximately
	if exp != float64(int(exp)) {
		// Simple approximation for 2.4 exponent
		frac := exp - float64(int(exp))
		result *= 1.0 + frac*(base-1.0)
	}
	return result
}

// SavePaletteToFile saves a palette to a TOML file.
func SavePaletteToFile(p *Palette, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate TOML content
	var sb strings.Builder

	sb.WriteString("# ")
	sb.WriteString(p.Name)
	sb.WriteString(" Color Palette\n")
	if p.Homepage != "" {
		sb.WriteString("# Source: ")
		sb.WriteString(p.Homepage)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	sb.WriteString("[palette]\n")
	sb.WriteString(fmt.Sprintf("name = %q\n", p.Name))
	sb.WriteString(fmt.Sprintf("variant = %q\n", p.Variant))
	if p.Author != "" {
		sb.WriteString(fmt.Sprintf("author = %q\n", p.Author))
	}
	if p.License != "" {
		sb.WriteString(fmt.Sprintf("license = %q\n", p.License))
	}
	if p.Homepage != "" {
		sb.WriteString(fmt.Sprintf("homepage = %q\n", p.Homepage))
	}
	if p.Description != "" {
		sb.WriteString(fmt.Sprintf("description = %q\n", p.Description))
	}

	sb.WriteString("\n# Raw Colors\n")
	sb.WriteString("[palette.colors]\n")

	// Sort color names for consistent output
	colorNames := make([]string, 0, len(p.Colors))
	for name := range p.Colors {
		colorNames = append(colorNames, name)
	}
	sort.Strings(colorNames)

	for _, name := range colorNames {
		sb.WriteString(fmt.Sprintf("%s = %q\n", name, p.Colors[name]))
	}

	sb.WriteString("\n# Semantic Colors\n")
	sb.WriteString("[palette.semantic]\n")

	// Sort semantic names for consistent output
	semanticNames := make([]string, 0, len(p.Semantic))
	for name := range p.Semantic {
		semanticNames = append(semanticNames, name)
	}
	sort.Strings(semanticNames)

	for _, name := range semanticNames {
		sb.WriteString(fmt.Sprintf("%s = %q\n", name, p.Semantic[name]))
	}

	if len(p.Components) > 0 {
		sb.WriteString("\n# Component Colors\n")
		sb.WriteString("[palette.components]\n")

		// Sort component names for consistent output
		componentNames := make([]string, 0, len(p.Components))
		for name := range p.Components {
			componentNames = append(componentNames, name)
		}
		sort.Strings(componentNames)

		for _, name := range componentNames {
			sb.WriteString(fmt.Sprintf("%s = %q\n", name, p.Components[name]))
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0o644) //nolint:gosec // palette files should be readable
}

// GetUserPalettesDir returns the user palettes directory.
func GetUserPalettesDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "markata-go", "palettes"), nil
}

// FetchLospecPalette is a convenience function that fetches a palette from Lospec.
func FetchLospecPalette(ctx context.Context, rawURL string) (*Palette, error) {
	client := NewLospecClient()
	return client.FetchPalette(ctx, rawURL)
}

// ParseLospecURL parses and validates a Lospec URL, returning the normalized URL.
func ParseLospecURL(rawURL string) (string, error) {
	// Parse and normalize the URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", ErrInvalidLospecURL
	}

	// Ensure it's a valid Lospec URL
	if u.Host != "lospec.com" {
		return "", ErrInvalidLospecURL
	}

	// Ensure HTTPS
	u.Scheme = "https"

	normalized := u.String()
	if err := ValidateLospecURL(normalized); err != nil {
		return "", err
	}

	return normalized, nil
}
