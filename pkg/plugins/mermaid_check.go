package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// MermaidDependencyInfo provides information about a dependency and installation instructions
type MermaidDependencyInfo struct {
	Mode                string
	IsInstalled         bool
	BinaryPath          string
	InstalledVersion    string
	InstallInstructions string
	FallbackSuggestion  string
}

// checkCLIDependency checks if mmdc (mermaid-cli) is installed
func checkCLIDependency(mmdc string) *MermaidDependencyInfo {
	info := &MermaidDependencyInfo{
		Mode: "cli",
		FallbackSuggestion: `To use client-side rendering instead, change your config:
  [markata-go.mermaid]
  mode = "client"`,
	}

	// Check if mmdc path is provided and exists
	if mmdc != "" {
		if _, err := os.Stat(mmdc); err == nil {
			info.IsInstalled = true
			info.BinaryPath = mmdc
			return info
		}
		// Provided path doesn't exist
		info.InstallInstructions = fmt.Sprintf(`mmdc not found at specified path: %s

Check the path and try again, or omit 'mmdc_path' to auto-detect.`, mmdc)
		return info
	}

	// Search for mmdc in PATH
	if path, err := exec.LookPath("mmdc"); err == nil {
		info.IsInstalled = true
		info.BinaryPath = path
		// Try to get version
		if out, err := exec.Command(path, "--version").Output(); err == nil {
			info.InstalledVersion = strings.TrimSpace(string(out))
		}
		return info
	}

	// Not found - provide install instructions
	info.InstallInstructions = `Missing dependency: @mermaid-js/mermaid-cli

Installation instructions:

1. Install Node.js v14+ from https://nodejs.org/
   (Check: node --version)

2. Install mermaid-cli globally:
   npm install -g @mermaid-js/mermaid-cli

3. Verify installation:
   mmdc --version

Or specify the path explicitly in your config:
  [markata-go.mermaid.cli]
  mmdc_path = "/path/to/mmdc"`

	return info
}

// checkChromiumDependency checks if Chrome/Chromium browser is installed
func checkChromiumDependency(browserPath string) *MermaidDependencyInfo {
	info := &MermaidDependencyInfo{
		Mode: "chromium",
		FallbackSuggestion: `To use client-side rendering instead, change your config:
  [markata-go.mermaid]
  mode = "client"`,
	}

	// Check provided path
	if browserPath != "" {
		if _, err := os.Stat(browserPath); err == nil {
			info.IsInstalled = true
			info.BinaryPath = browserPath
			return info
		}
		// Provided path doesn't exist
		info.InstallInstructions = fmt.Sprintf(`Browser not found at specified path: %s

Check the path and try again, or omit 'browser_path' to auto-detect.`, browserPath)
		return info
	}

	// Try common locations for different OSes
	commonPaths := []string{
		// Linux
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		// macOS
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		// Windows (unlikely in this context, but include for completeness)
		"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
		"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			info.IsInstalled = true
			info.BinaryPath = path
			return info
		}
	}

	// Not found
	info.InstallInstructions = `Missing dependency: Chromium/Chrome browser

Installation instructions depend on your OS:

Linux (Debian/Ubuntu):
  sudo apt-get install chromium-browser

Linux (Fedora/RHEL):
  sudo dnf install chromium

macOS:
  brew install chromium

Windows:
  choco install chromium

Or download from: https://www.chromium.org/getting-involved/download-chromium

Specify the path in your config:
  [markata-go.mermaid.chromium]
  browser_path = "/path/to/chromium"`

	return info
}

// ValidateMermaidMode validates the mermaid rendering mode and checks dependencies
func ValidateMermaidMode(mode string, cliConfig *CLIConfig, chromiumConfig *ChromiumConfig) *MermaidDependencyInfo {
	switch mode {
	case "client":
		// No dependencies needed for client-side rendering
		return &MermaidDependencyInfo{
			Mode:        "client",
			IsInstalled: true,
		}

	case "cli":
		if cliConfig == nil {
			cliConfig = &CLIConfig{}
		}
		return checkCLIDependency(cliConfig.MMDCPath)

	case "chromium":
		if chromiumConfig == nil {
			chromiumConfig = &ChromiumConfig{}
		}
		return checkChromiumDependency(chromiumConfig.BrowserPath)

	default:
		return &MermaidDependencyInfo{
			Mode:        "unknown",
			IsInstalled: false,
			InstallInstructions: fmt.Sprintf(`Invalid mermaid rendering mode: %q

Valid modes are:
  - "client"   (browser-based, no dependencies)
  - "cli"      (npm mmdc command-line tool)
  - "chromium" (Chrome DevTools Protocol)`, mode),
		}
	}
}

// CLIConfig is a helper for CLI renderer configuration
type CLIConfig struct {
	MMDCPath  string
	ExtraArgs string
}

// ChromiumConfig is a helper for Chromium renderer configuration
type ChromiumConfig struct {
	BrowserPath   string
	Timeout       int
	MaxConcurrent int
}
