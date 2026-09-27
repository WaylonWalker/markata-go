// Package serveopen builds commands for opening local source and preview URLs.
// It never sends a path through a shell.
package serveopen

import (
	"fmt"
	"net"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// EditorCommand builds the user's editor command. Terminal clients can pass it
// to Bubble Tea's ExecProcess so the alternate screen is suspended cleanly.
func EditorCommand(path string, line int) (*exec.Cmd, error) {
	if path == "" {
		return nil, fmt.Errorf("source path is required")
	}
	command := strings.TrimSpace(os.Getenv("EDITOR"))
	if command == "" {
		command = strings.TrimSpace(os.Getenv("VISUAL"))
	}
	if command == "" {
		for _, candidate := range []string{"vim", "nano"} {
			if _, err := exec.LookPath(candidate); err == nil {
				command = candidate
				break
			}
		}
	}
	if command == "" {
		return nil, fmt.Errorf("set $EDITOR to open source files")
	}
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("$EDITOR is empty")
	}
	name := filepath.Base(parts[0])
	args := append([]string(nil), parts[1:]...)
	switch name {
	case "vim", "nvim", "vi", "emacs", "nano":
		if line > 0 {
			args = append(args, "+"+strconv.Itoa(line))
		}
		args = append(args, path)
	case "code", "codium":
		if line > 0 {
			args = append(args, "--goto", path+":"+strconv.Itoa(line))
		} else {
			args = append(args, path)
		}
	default:
		args = append(args, path)
	}
	return exec.Command(parts[0], args...), nil // #nosec G204 -- executable and arguments come from the user's explicit editor configuration.
}

// BrowserCommand builds a preview command, honoring $BROWSER when set.
func BrowserCommand(url string) (*exec.Cmd, error) {
	parsed, err := neturl.Parse(url)
	if err != nil || parsed.Scheme != "http" || parsed.Opaque != "" || parsed.User != nil || !isLoopbackHost(parsed.Hostname()) {
		return nil, fmt.Errorf("preview URL must use the local serve listener")
	}
	if browser := strings.TrimSpace(os.Getenv("BROWSER")); browser != "" {
		parts := strings.Fields(browser)
		return exec.Command(parts[0], append(parts[1:], url)...), nil // #nosec G204 -- executable comes from the user's browser setting; url was restricted to localhost above.
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url), nil
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url), nil
	default:
		return exec.Command("xdg-open", url), nil
	}
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// IsTerminalEditor reports whether a web action needs a terminal client to run
// the chosen editor. GUI editors can be started from the local web client.
func IsTerminalEditor(cmd *exec.Cmd) bool {
	if cmd == nil || len(cmd.Args) == 0 {
		return false
	}
	switch filepath.Base(cmd.Args[0]) {
	case "vim", "nvim", "vi", "nano", "emacs":
		return true
	default:
		return false
	}
}
