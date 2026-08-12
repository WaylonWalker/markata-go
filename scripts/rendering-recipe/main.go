// Command rendering-recipe validates and compiles the canonical recipe bundle.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/WaylonWalker/markata-go/pkg/renderingrecipe"
)

func main() {
	check := flag.Bool("check", false, "validate the compiler and deterministic output")
	out := flag.String("out", "", "write a concrete bundle directory")
	flag.Parse()
	theme, err := renderingrecipe.LoadCanonicalTheme()
	if err != nil {
		fail(err)
	}
	bundle, err := renderingrecipe.Compile(theme)
	if err != nil {
		fail(err)
	}
	second, err := renderingrecipe.Compile(theme)
	if err != nil {
		fail(err)
	}
	if bundle.Manifest.RecipeHash != second.Manifest.RecipeHash {
		fail(fmt.Errorf("compiler output is not deterministic"))
	}
	if *out != "" {
		writeBundle(*out, bundle)
	}
	if *check {
		fmt.Printf("rendering recipe compiler OK: semantic=%s recipe=%s assets=%d\n", bundle.Manifest.SemanticHash, bundle.Manifest.RecipeHash, len(bundle.Assets))
		return
	}
	data, err := json.MarshalIndent(bundle.Manifest, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(data))
}

func writeBundle(root string, bundle renderingrecipe.Bundle) {
	if err := os.MkdirAll(root, 0755); err != nil {
		fail(err)
	}
	data, err := json.MarshalIndent(bundle.Manifest, "", "  ")
	if err != nil {
		fail(err)
	}
	if err = os.WriteFile(filepath.Join(root, "manifest.json"), append(data, '\n'), 0644); err != nil {
		fail(err)
	}
	for path, data := range bundle.Assets {
		target := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			fail(err)
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			fail(err)
		}
	}
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
