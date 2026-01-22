package palettes

import (
	"testing"
)

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantR   uint8
		wantG   uint8
		wantB   uint8
		wantErr bool
	}{
		{"6-digit with hash", "#ff0000", 255, 0, 0, false},
		{"6-digit without hash", "ff0000", 255, 0, 0, false},
		{"3-digit with hash", "#f00", 255, 0, 0, false},
		{"3-digit without hash", "f00", 255, 0, 0, false},
		{"lowercase", "#aabbcc", 170, 187, 204, false},
		{"uppercase", "#AABBCC", 170, 187, 204, false},
		{"mixed case", "#AaBbCc", 170, 187, 204, false},
		{"8-digit (with alpha)", "#ff0000ff", 255, 0, 0, false},
		{"4-digit (with alpha)", "#f00f", 255, 0, 0, false},
		{"catppuccin mocha base", "#1e1e2e", 30, 30, 46, false},
		{"catppuccin mocha text", "#cdd6f4", 205, 214, 244, false},
		{"invalid - too short", "#ff", 0, 0, 0, true},
		{"invalid - too long", "#ff00ff00ff", 0, 0, 0, true},
		{"invalid - non-hex", "#gggggg", 0, 0, 0, true},
		{"invalid - empty", "", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := ParseHexColor(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHexColor(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if c.R != tt.wantR || c.G != tt.wantG || c.B != tt.wantB {
					t.Errorf("ParseHexColor(%q) = (%d, %d, %d), want (%d, %d, %d)",
						tt.input, c.R, c.G, c.B, tt.wantR, tt.wantG, tt.wantB)
				}
			}
		})
	}
}

func TestColor_Hex(t *testing.T) {
	tests := []struct {
		name  string
		color Color
		want  string
	}{
		{"red", Color{255, 0, 0}, "#ff0000"},
		{"green", Color{0, 255, 0}, "#00ff00"},
		{"blue", Color{0, 0, 255}, "#0000ff"},
		{"white", Color{255, 255, 255}, "#ffffff"},
		{"black", Color{0, 0, 0}, "#000000"},
		{"gray", Color{128, 128, 128}, "#808080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.color.Hex()
			if got != tt.want {
				t.Errorf("Color.Hex() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestColor_RelativeLuminance(t *testing.T) {
	tests := []struct {
		name    string
		color   Color
		wantMin float64
		wantMax float64
	}{
		{"white", Color{255, 255, 255}, 0.99, 1.01},
		{"black", Color{0, 0, 0}, -0.01, 0.01},
		{"red", Color{255, 0, 0}, 0.20, 0.22},
		{"green", Color{0, 255, 0}, 0.71, 0.73},
		{"blue", Color{0, 0, 255}, 0.07, 0.08},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.color.RelativeLuminance()
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("Color.RelativeLuminance() = %v, want between %v and %v",
					got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestContrastRatio(t *testing.T) {
	tests := []struct {
		name    string
		fg      Color
		bg      Color
		wantMin float64
		wantMax float64
	}{
		{"black on white", Color{0, 0, 0}, Color{255, 255, 255}, 20.9, 21.1},
		{"white on black", Color{255, 255, 255}, Color{0, 0, 0}, 20.9, 21.1},
		{"same color", Color{128, 128, 128}, Color{128, 128, 128}, 0.99, 1.01},
		{"light gray on white", Color{200, 200, 200}, Color{255, 255, 255}, 1.0, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContrastRatio(tt.fg, tt.bg)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("ContrastRatio() = %v, want between %v and %v",
					got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestMeetsWCAG(t *testing.T) {
	tests := []struct {
		name        string
		ratio       float64
		level       WCAGLevel
		isLargeText bool
		want        bool
	}{
		{"AA normal text - pass", 4.5, WCAGLevelAA, false, true},
		{"AA normal text - fail", 4.4, WCAGLevelAA, false, false},
		{"AA large text - pass", 3.0, WCAGLevelAA, true, true},
		{"AA large text - fail", 2.9, WCAGLevelAA, true, false},
		{"AAA normal text - pass", 7.0, WCAGLevelAAA, false, true},
		{"AAA normal text - fail", 6.9, WCAGLevelAAA, false, false},
		{"AAA large text - pass", 4.5, WCAGLevelAAA, true, true},
		{"AAA large text - fail", 4.4, WCAGLevelAAA, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MeetsWCAG(tt.ratio, tt.level, tt.isLargeText)
			if got != tt.want {
				t.Errorf("MeetsWCAG(%v, %v, %v) = %v, want %v",
					tt.ratio, tt.level, tt.isLargeText, got, tt.want)
			}
		})
	}
}

func TestPassedLevels(t *testing.T) {
	tests := []struct {
		name        string
		ratio       float64
		isLargeText bool
		wantLen     int
	}{
		{"21:1 contrast passes all", 21.0, false, 3},
		{"7:1 contrast passes AA and AAA", 7.0, false, 3},
		{"4.5:1 contrast passes A and AA", 4.5, false, 2},
		{"3:1 contrast passes only A", 3.0, false, 1},
		{"2:1 contrast passes none", 2.0, false, 0},
		{"4.5:1 large text passes all", 4.5, true, 3},
		{"3:1 large text passes A and AA", 3.0, true, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PassedLevels(tt.ratio, tt.isLargeText)
			if len(got) != tt.wantLen {
				t.Errorf("PassedLevels(%v, %v) returned %d levels, want %d",
					tt.ratio, tt.isLargeText, len(got), tt.wantLen)
			}
		})
	}
}
