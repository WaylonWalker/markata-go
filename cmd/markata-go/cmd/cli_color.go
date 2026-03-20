package cmd

import (
	"fmt"

	"github.com/WaylonWalker/markata-go/pkg/palettes"
)

const (
	ansiBlueBold  = "\033[1;34m"
	ansiCyan      = "\033[36m"
	ansiGreenBold = "\033[1;32m"
	ansiMagenta   = "\033[35m"
	ansiReset     = "\033[0m"
	ansiYellow    = "\033[33m"
)

func colorizeOutput(text, color string) string {
	if !colorEnabledOnOutput() {
		return text
	}
	if rgb := ansiTrueColor(color); rgb != "" {
		return rgb + text + ansiReset
	}
	return color + text + ansiReset
}

func ansiTrueColor(hex string) string {
	if len(hex) == 0 || hex[0] != '#' {
		return ""
	}
	parsed, err := palettes.ParseHexColor(hex)
	if err != nil {
		return ""
	}
	r, g, b, _ := parsed.RGBA()
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r>>8, g>>8, b>>8)
}
