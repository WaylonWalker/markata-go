package plugins

import (
	"context"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
	"go.abhg.dev/goldmark/mermaid"
	"go.abhg.dev/goldmark/mermaid/mermaidcdp"
)

// mermaidRenderer is an interface for rendering Mermaid diagrams in different modes
type mermaidRenderer interface {
	// render processes a mermaid diagram code and returns the rendered output
	// For client mode: returns the diagram code wrapped in mermaid markers
	// For cli/chromium modes: returns the SVG output
	render(diagramCode string) (string, error)
	// close cleans up any resources (e.g., browser connections)
	close() error
}

// clientRenderer renders diagrams client-side via JavaScript
type clientRenderer struct {
	config models.MermaidConfig
}

func (r *clientRenderer) render(diagramCode string) (string, error) {
	// Client-side rendering: just return the diagram code, the script will render it
	return diagramCode, nil
}

func (r *clientRenderer) close() error {
	return nil
}

// cliRenderer renders diagrams using the mmdc CLI tool
type cliRenderer struct {
	config models.MermaidConfig
}

func (r *cliRenderer) render(diagramCode string) (string, error) {
	mmdc := r.config.CLIConfig.MMDCPath
	if mmdc == "" {
		var err error
		mmdc, err = exec.LookPath("mmdc")
		if err != nil {
			return "", fmt.Errorf("mmdc binary not found: %w", err)
		}
	}

	// Create temporary directory for diagram files
	tmpDir, err := os.MkdirTemp("", "markata-mermaid-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write diagram code to temp file
	inputFile := filepath.Join(tmpDir, "diagram.mmd")
	outputFile := filepath.Join(tmpDir, "diagram.svg")

	if err := os.WriteFile(inputFile, []byte(diagramCode), 0o600); err != nil {
		return "", fmt.Errorf("failed to write diagram file: %w", err)
	}

	// Run mmdc to render the diagram
	cmd := exec.Command(mmdc, "-i", inputFile, "-o", outputFile)
	if r.config.CLIConfig.ExtraArgs != "" {
		cmd.Args = append(cmd.Args, strings.Fields(r.config.CLIConfig.ExtraArgs)...)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("mmdc rendering failed: %w (output: %s)", err, string(output))
	}

	// Read the generated SVG
	svgBytes, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read generated SVG: %w", err)
	}

	return string(svgBytes), nil
}

func (r *cliRenderer) close() error {
	return nil
}

// chromiumRenderer renders diagrams using Chrome DevTools Protocol via mermaidcdp
type chromiumRenderer struct {
	config    models.MermaidConfig
	compiler  *mermaidcdp.Compiler
	once      sync.Once
	initErr   error
	semaphore chan struct{} // limits concurrent renders
}

func (r *chromiumRenderer) ensureCompiler() error {
	var initErr error
	r.once.Do(func() {
		// Create compiler config with the specified theme
		cfg := &mermaidcdp.Config{
			Theme: r.config.Theme,
		}

		var err error
		r.compiler, err = mermaidcdp.New(cfg)
		if err != nil {
			initErr = err
			return
		}

		// Initialize the semaphore for concurrent renders
		maxConcurrent := r.config.ChromiumConfig.MaxConcurrent
		if maxConcurrent <= 0 {
			maxConcurrent = 4
		}
		r.semaphore = make(chan struct{}, maxConcurrent)
	})
	if initErr != nil {
		return initErr
	}
	return r.initErr
}

func (r *chromiumRenderer) render(diagramCode string) (string, error) {
	if err := r.ensureCompiler(); err != nil {
		return "", err
	}

	// Acquire semaphore slot
	r.semaphore <- struct{}{}
	defer func() { <-r.semaphore }()

	timeout := r.config.ChromiumConfig.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Compile the diagram via mermaidcdp
	req := &mermaid.CompileRequest{
		Source: diagramCode,
	}

	resp, err := r.compiler.Compile(ctx, req)
	if err != nil {
		return "", fmt.Errorf("mermaidcdp rendering failed: %w", err)
	}

	// Clean up the SVG output - unescape any HTML entities that may have been added
	svgOutput := html.UnescapeString(resp.SVG)
	return strings.TrimSpace(svgOutput), nil
}

func (r *chromiumRenderer) close() error {
	if r.compiler != nil {
		return r.compiler.Close()
	}
	return nil
}

// newMermaidRenderer creates the appropriate renderer based on the config mode
func newMermaidRenderer(config models.MermaidConfig) (mermaidRenderer, error) {
	switch config.Mode {
	case "client":
		return &clientRenderer{config: config}, nil

	case "cli":
		// Validate CLI dependencies
		info := checkCLIDependency(config.CLIConfig.MMDCPath)
		if !info.IsInstalled {
			err := models.NewMermaidRenderError("", config.Mode, "mmdc binary not found", nil)
			err.Suggestion = info.InstallInstructions + "\n\n" + info.FallbackSuggestion
			return nil, err
		}
		return &cliRenderer{config: config}, nil

	case "chromium":
		// Validate Chromium dependencies
		info := checkChromiumDependency(config.ChromiumConfig.BrowserPath)
		if !info.IsInstalled {
			err := models.NewMermaidRenderError("", config.Mode, "browser not found", nil)
			err.Suggestion = info.InstallInstructions + "\n\n" + info.FallbackSuggestion
			return nil, err
		}
		return &chromiumRenderer{config: config}, nil

	default:
		return nil, fmt.Errorf("invalid mermaid rendering mode: %q", config.Mode)
	}
}
