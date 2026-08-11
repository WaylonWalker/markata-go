// Command generate-rendering-contract projects the language-neutral contract to
// the Go and browser consumers. Run it from the markata-go repository root.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/WaylonWalker/markata-go/pkg/renderingcontract"
)

func main() {
	check := flag.Bool("check", false, "check generated files without writing")
	resolveFixtures := flag.Bool("resolve-fixtures", false, "write resolved shared render-plan fixtures as JSON")
	flag.Parse()
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	source := filepath.Join(root, "spec", "rendering-contract", "contract-v1.json")
	raw, err := os.ReadFile(source)
	if err != nil {
		panic(err)
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		panic(err)
	}
	if *resolveFixtures {
		fixturesPath := filepath.Join(root, "spec", "rendering-contract", "render-plan-fixtures.json")
		fixtureRaw, err := os.ReadFile(fixturesPath)
		if err != nil {
			panic(err)
		}
		var fixtures renderingcontract.RenderPlanFixtures
		if err := json.Unmarshal(fixtureRaw, &fixtures); err != nil {
			panic(err)
		}
		results := make([]map[string]any, 0, len(fixtures.Fixtures))
		for _, fixture := range fixtures.Fixtures {
			state, plan, _, err := renderingcontract.ResolveFixture(fixture)
			if err != nil {
				panic(err)
			}
			results = append(results, map[string]any{"id": fixture.ID, "normalized": state, "plan": plan})
		}
		encoded, err := json.Marshal(results)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(encoded))
		return
	}
	pretty, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	pretty = append(pretty, '\n')
	jsonTargets := []string{filepath.Join(root, "pkg", "renderingcontract", "contract-v1.json")}
	jsTargets := map[string][]byte{
		filepath.Join(root, "..", "md.waylonwalker.com", "src", "rendering-contract.generated.js"):     append([]byte("// Generated. Do not edit.\nexport const RENDERING_CONTRACT = Object.freeze("), append(pretty, []byte(");\nexport const CONTRACT_PALETTES = RENDERING_CONTRACT.palettes;\nexport const CONTRACT_ENUMS = RENDERING_CONTRACT.enums;\n")...)...),
		filepath.Join(root, "..", "themes.waylonwalker.com", "src", "rendering-contract.generated.js"): append([]byte("// Generated. Do not edit.\nwindow.renderingContract = "), append(pretty, []byte(";\n")...)...),
	}
	for _, target := range jsonTargets {
		if err := project(target, pretty, *check); err != nil {
			panic(err)
		}
	}
	for target, data := range jsTargets {
		if *check {
			if _, err := os.Stat(target); os.IsNotExist(err) {
				// The browser projections live in sibling repositories during the
				// coordinated workspace, but are not present in a standalone Go
				// checkout. CI still verifies the Go projection here; the full
				// workspace check verifies all three when siblings are available.
				fmt.Printf("skipping absent sibling projection %s\n", target)
				continue
			}
		}
		if err := project(target, data, *check); err != nil {
			panic(err)
		}
	}
	if *check {
		fmt.Println("rendering contract generated artifacts are current")
	} else {
		fmt.Println("rendering contract generated artifacts updated")
	}
}

func project(path string, expected []byte, check bool) error {
	actual, err := os.ReadFile(path)
	if check {
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if !bytes.Equal(actual, expected) {
			return fmt.Errorf("%s is stale", path)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, expected, 0o644)
}
