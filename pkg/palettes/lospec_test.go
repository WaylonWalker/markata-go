package palettes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateLospecURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid URL",
			url:     "https://lospec.com/palette-list/cheese-palette.txt",
			wantErr: false,
		},
		{
			name:    "valid URL with dashes",
			url:     "https://lospec.com/palette-list/some-cool-palette.txt",
			wantErr: false,
		},
		{
			name:    "valid URL with numbers",
			url:     "https://lospec.com/palette-list/palette123.txt",
			wantErr: false,
		},
		{
			name:    "valid URL with underscores",
			url:     "https://lospec.com/palette-list/my_palette.txt",
			wantErr: false,
		},
		{
			name:    "invalid - wrong domain",
			url:     "https://example.com/palette-list/test.txt",
			wantErr: true,
		},
		{
			name:    "invalid - wrong path",
			url:     "https://lospec.com/palettes/test.txt",
			wantErr: true,
		},
		{
			name:    "invalid - no .txt extension",
			url:     "https://lospec.com/palette-list/test",
			wantErr: true,
		},
		{
			name:    "invalid - http instead of https",
			url:     "http://lospec.com/palette-list/test.txt",
			wantErr: true,
		},
		{
			name:    "invalid - empty name",
			url:     "https://lospec.com/palette-list/.txt",
			wantErr: true,
		},
		{
			name:    "invalid - special characters",
			url:     "https://lospec.com/palette-list/test@palette.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLospecURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLospecURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestExtractPaletteNameFromURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name:    "simple name",
			url:     "https://lospec.com/palette-list/cheese.txt",
			want:    "Cheese",
			wantErr: false,
		},
		{
			name:    "kebab-case name",
			url:     "https://lospec.com/palette-list/cheese-palette.txt",
			want:    "Cheese Palette",
			wantErr: false,
		},
		{
			name:    "multi-word name",
			url:     "https://lospec.com/palette-list/super-cool-retro-theme.txt",
			want:    "Super Cool Retro Theme",
			wantErr: false,
		},
		{
			name:    "invalid URL",
			url:     "https://example.com/test.txt",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractPaletteNameFromURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractPaletteNameFromURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractPaletteNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestParseLospecColors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:    "colors with hash prefix",
			input:   "#ff0000\n#00ff00\n#0000ff\n",
			want:    []string{"#ff0000", "#00ff00", "#0000ff"},
			wantErr: false,
		},
		{
			name:    "colors without hash prefix",
			input:   "ff0000\n00ff00\n0000ff\n",
			want:    []string{"#ff0000", "#00ff00", "#0000ff"},
			wantErr: false,
		},
		{
			name:    "mixed prefix",
			input:   "#ff0000\n00ff00\n#0000ff\n",
			want:    []string{"#ff0000", "#00ff00", "#0000ff"},
			wantErr: false,
		},
		{
			name:    "with empty lines",
			input:   "#ff0000\n\n#00ff00\n\n#0000ff\n",
			want:    []string{"#ff0000", "#00ff00", "#0000ff"},
			wantErr: false,
		},
		{
			name:    "with whitespace",
			input:   "  #ff0000  \n  00ff00\n#0000ff  \n",
			want:    []string{"#ff0000", "#00ff00", "#0000ff"},
			wantErr: false,
		},
		{
			name:    "short hex colors",
			input:   "#f00\n#0f0\n#00f\n",
			want:    []string{"#ff0000", "#00ff00", "#0000ff"},
			wantErr: false,
		},
		{
			name:    "uppercase colors",
			input:   "#FF0000\n#00FF00\n#0000FF\n",
			want:    []string{"#ff0000", "#00ff00", "#0000ff"},
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   "",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "only whitespace",
			input:   "   \n\n   \n",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid colors skipped",
			input:   "#ff0000\ninvalid\n#00ff00\nnotacolor\n#0000ff\n",
			want:    []string{"#ff0000", "#00ff00", "#0000ff"},
			wantErr: false,
		},
		{
			name:    "ARGB format (Paint.NET)",
			input:   "FFff0000\nFF00ff00\nFF0000ff\n",
			want:    []string{"#ff0000", "#00ff00", "#0000ff"},
			wantErr: false,
		},
		{
			name: "Paint.NET palette with comments",
			input: `;paint.net Palette File
;Downloaded from Lospec.com/palette-list
;Colors: 3
FFff0000
FF00ff00
FF0000ff
`,
			want:    []string{"#ff0000", "#00ff00", "#0000ff"},
			wantErr: false,
		},
		{
			name:    "only comments",
			input:   ";comment\n;another comment\n",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLospecColors(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLospecColors() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("parseLospecColors() got %d colors, want %d", len(got), len(tt.want))
					return
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("parseLospecColors()[%d] = %q, want %q", i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestCreatePaletteFromColors(t *testing.T) {
	tests := []struct {
		name        string
		paletteName string
		colors      []string
		wantVariant Variant
		wantSemKeys []string
	}{
		{
			name:        "dark palette (dark colors)",
			paletteName: "Test Dark",
			colors:      []string{"#1a1a1a", "#2a2a2a", "#ff6b6b", "#e0e0e0", "#ffffff"},
			wantVariant: VariantDark,
			wantSemKeys: []string{"bg-primary", "text-primary"},
		},
		{
			name:        "light palette (light colors)",
			paletteName: "Test Light",
			colors:      []string{"#ffffff", "#f0f0f0", "#eeeeee", "#dddddd", "#666666"},
			wantVariant: VariantLight,
			wantSemKeys: []string{"bg-primary", "text-primary"},
		},
		{
			name:        "minimal palette (2 colors)",
			paletteName: "Minimal",
			colors:      []string{"#000000", "#ffffff"},
			wantVariant: VariantDark,
			wantSemKeys: []string{"bg-primary", "text-primary"},
		},
		{
			name:        "single color",
			paletteName: "Single",
			colors:      []string{"#888888"},
			wantVariant: VariantDark,
			wantSemKeys: []string{"bg-primary", "text-primary"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := CreatePaletteFromColors(tt.paletteName, tt.colors, "https://lospec.com/test")

			if p.Name != tt.paletteName {
				t.Errorf("Name = %q, want %q", p.Name, tt.paletteName)
			}

			if p.Variant != tt.wantVariant {
				t.Errorf("Variant = %q, want %q", p.Variant, tt.wantVariant)
			}

			// Check that all colors are in the palette
			if len(p.Colors) != len(tt.colors) {
				t.Errorf("Colors count = %d, want %d", len(p.Colors), len(tt.colors))
			}

			// Check that semantic keys exist
			for _, key := range tt.wantSemKeys {
				if _, ok := p.Semantic[key]; !ok {
					t.Errorf("Missing semantic key: %s", key)
				}
			}

			// Validate the palette
			errs := p.Validate()
			if len(errs) > 0 {
				t.Errorf("Palette validation failed: %v", errs)
			}
		})
	}
}

func TestLospecClient_FetchPalette_MockServer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Return a simple palette
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("#1a1b26\n#24283b\n#7aa2f7\n#bb9af7\n#c0caf5\n")) //nolint:errcheck // test code
	}))
	defer server.Close()

	// Create client with custom HTTP client that redirects to our mock server
	client := &LospecClient{
		httpClient: server.Client(),
		cacheDir:   "", // Disable caching for this test
		userAgent:  "test-agent",
	}

	// We can't directly test against lospec.com, so we'll test the parsing logic
	colors, err := client.fetchColors(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetchColors() error = %v", err)
	}

	if len(colors) != 5 {
		t.Errorf("fetchColors() returned %d colors, want 5", len(colors))
	}

	expected := []string{"#1a1b26", "#24283b", "#7aa2f7", "#bb9af7", "#c0caf5"}
	for i, color := range colors {
		if color != expected[i] {
			t.Errorf("color[%d] = %q, want %q", i, color, expected[i])
		}
	}
}

func TestLospecClient_Cache(t *testing.T) {
	// Create temp cache directory
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	client := NewLospecClient(
		WithLospecCacheDir(cacheDir),
	)

	// Test URL
	testURL := "https://lospec.com/palette-list/test-palette.txt"
	colors := []string{"#ff0000", "#00ff00", "#0000ff"}

	// Save to cache
	err := client.saveToCache(testURL, colors)
	if err != nil {
		t.Fatalf("saveToCache() error = %v", err)
	}

	// Verify cache directory was created
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("Cache directory was not created")
	}

	// Load from cache
	cached, err := client.loadFromCache(testURL)
	if err != nil {
		t.Fatalf("loadFromCache() error = %v", err)
	}

	if cached.Name != "Test Palette" {
		t.Errorf("Cached palette name = %q, want %q", cached.Name, "Test Palette")
	}

	if len(cached.Colors) != 3 {
		t.Errorf("Cached palette has %d colors, want 3", len(cached.Colors))
	}
}

func TestLospecClient_Options(t *testing.T) {
	// Test WithLospecTimeout
	client := NewLospecClient(
		WithLospecTimeout(60 * time.Second),
	)
	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want %v", client.httpClient.Timeout, 60*time.Second)
	}

	// Test WithLospecUserAgent
	client = NewLospecClient(
		WithLospecUserAgent("custom-agent/1.0"),
	)
	if client.userAgent != "custom-agent/1.0" {
		t.Errorf("UserAgent = %q, want %q", client.userAgent, "custom-agent/1.0")
	}

	// Test WithLospecCacheDir
	client = NewLospecClient(
		WithLospecCacheDir("/custom/cache"),
	)
	if client.cacheDir != "/custom/cache" {
		t.Errorf("CacheDir = %q, want %q", client.cacheDir, "/custom/cache")
	}
}

func TestSavePaletteToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-palette.toml")

	p := NewPalette("Test Palette", VariantDark)
	p.Homepage = "https://lospec.com/palette-list/test-palette.txt"
	p.Description = "Test description"
	p.Colors["color0"] = "#ff0000"
	p.Colors["color1"] = "#00ff00"
	p.Semantic["bg-primary"] = "color0"
	p.Semantic["text-primary"] = "color1"

	err := SavePaletteToFile(p, outputPath)
	if err != nil {
		t.Fatalf("SavePaletteToFile() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Output file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Verify key content
	contentStr := string(content)
	if !strings.Contains(contentStr, `name = "Test Palette"`) {
		t.Error("Output missing palette name")
	}
	if !strings.Contains(contentStr, `variant = "dark"`) {
		t.Error("Output missing variant")
	}
	if !strings.Contains(contentStr, `color0 = "#ff0000"`) {
		t.Error("Output missing color0")
	}
	if !strings.Contains(contentStr, `bg-primary = "color0"`) {
		t.Error("Output missing bg-primary semantic")
	}

	// Try to load it back
	loaded, err := LoadFromFile(outputPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if loaded.Name != p.Name {
		t.Errorf("Loaded Name = %q, want %q", loaded.Name, p.Name)
	}
	if loaded.Variant != p.Variant {
		t.Errorf("Loaded Variant = %q, want %q", loaded.Variant, p.Variant)
	}
}

func TestGetUserPalettesDir(t *testing.T) {
	dir, err := GetUserPalettesDir()
	if err != nil {
		t.Fatalf("GetUserPalettesDir() error = %v", err)
	}

	if dir == "" {
		t.Error("GetUserPalettesDir() returned empty string")
	}

	// Should contain markata-go/palettes
	if !strings.Contains(dir, "markata-go") {
		t.Errorf("GetUserPalettesDir() = %q, should contain 'markata-go'", dir)
	}
	if !strings.Contains(dir, "palettes") {
		t.Errorf("GetUserPalettesDir() = %q, should contain 'palettes'", dir)
	}
}

func TestParseLospecURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name:    "valid HTTPS URL",
			url:     "https://lospec.com/palette-list/cheese-palette.txt",
			want:    "https://lospec.com/palette-list/cheese-palette.txt",
			wantErr: false,
		},
		{
			name:    "invalid domain",
			url:     "https://example.com/palette-list/test.txt",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid path",
			url:     "https://lospec.com/other/test.txt",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLospecURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLospecURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseLospecURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestHexToRGB(t *testing.T) {
	tests := []struct {
		hex     string
		r, g, b uint8
	}{
		{"#ff0000", 255, 0, 0},
		{"#00ff00", 0, 255, 0},
		{"#0000ff", 0, 0, 255},
		{"#ffffff", 255, 255, 255},
		{"#000000", 0, 0, 0},
		{"#f00", 255, 0, 0},   // Short form
		{"ff0000", 255, 0, 0}, // Without hash
	}

	for _, tt := range tests {
		t.Run(tt.hex, func(t *testing.T) {
			r, g, b := hexToRGB(tt.hex)
			if r != tt.r || g != tt.g || b != tt.b {
				t.Errorf("hexToRGB(%q) = (%d, %d, %d), want (%d, %d, %d)",
					tt.hex, r, g, b, tt.r, tt.g, tt.b)
			}
		})
	}
}

func TestSaturation(t *testing.T) {
	tests := []struct {
		hex  string
		want float64
	}{
		{"#ff0000", 1.0}, // Full red, fully saturated
		{"#00ff00", 1.0}, // Full green, fully saturated
		{"#0000ff", 1.0}, // Full blue, fully saturated
		{"#ffffff", 0.0}, // White, no saturation
		{"#000000", 0.0}, // Black, no saturation
		{"#808080", 0.0}, // Gray, no saturation
	}

	for _, tt := range tests {
		t.Run(tt.hex, func(t *testing.T) {
			got := saturation(tt.hex)
			if got != tt.want {
				t.Errorf("saturation(%q) = %v, want %v", tt.hex, got, tt.want)
			}
		})
	}
}

func TestRelativeLuminance(t *testing.T) {
	// Just verify that white is brighter than black
	whiteLum := relativeLuminance("#ffffff")
	blackLum := relativeLuminance("#000000")

	if whiteLum <= blackLum {
		t.Errorf("White luminance (%v) should be greater than black (%v)", whiteLum, blackLum)
	}

	// White should be close to 1.0
	if whiteLum < 0.9 {
		t.Errorf("White luminance = %v, should be close to 1.0", whiteLum)
	}

	// Black should be close to 0.0
	if blackLum > 0.1 {
		t.Errorf("Black luminance = %v, should be close to 0.0", blackLum)
	}
}

func TestLospecFetchError(t *testing.T) {
	// Test error with status code
	err1 := NewLospecFetchError("https://lospec.com/test", 404, "not found", nil)
	if !strings.Contains(err1.Error(), "404") {
		t.Errorf("Error message should contain status code: %s", err1.Error())
	}

	// Test error without status code
	err2 := NewLospecFetchError("https://lospec.com/test", 0, "connection failed", nil)
	if strings.Contains(err2.Error(), "HTTP 0") {
		t.Errorf("Error message should not contain 'HTTP 0': %s", err2.Error())
	}

	// Test Unwrap
	underlying := ErrLospecFetchFailed
	err3 := NewLospecFetchError("test", 500, "error", underlying)
	if !errors.Is(err3.Unwrap(), underlying) {
		t.Error("Unwrap should return underlying error")
	}
}
