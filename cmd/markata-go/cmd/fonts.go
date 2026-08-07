package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/config"
	"github.com/WaylonWalker/markata-go/pkg/fontpacks"
	"github.com/spf13/cobra"
)

var fontsCmd = &cobra.Command{Use: "fonts", Short: "Inspect and maintain Markata font packs"}
var fontsListCmd = &cobra.Command{Use: "list", Short: "List available font sources", RunE: runFontsList}
var fontsPacksCmd = &cobra.Command{Use: "packs", Short: "List available font packs", RunE: runFontsPacks}
var fontsShowCmd = &cobra.Command{Use: "show <font-or-pack>", Args: cobra.ExactArgs(1), RunE: runFontsShow}
var fontsDoctorCmd = &cobra.Command{Use: "doctor", Short: "Check optional FontTools support", RunE: runFontsDoctor}
var fontsVerifyCmd = &cobra.Command{Use: "verify [pack]", Short: "Validate the local font catalog", Args: cobra.MaximumNArgs(1), RunE: runFontsVerify}
var fontsReportCmd = &cobra.Command{Use: "report", Short: "Report selected pack size", RunE: runFontsReport}
var fontsLicensesCmd = &cobra.Command{Use: "licenses", Short: "List catalog license records", RunE: runFontsLicenses}

func init() {
	rootCmd.AddCommand(fontsCmd)
	fontsCmd.AddCommand(fontsListCmd, fontsPacksCmd, fontsShowCmd, fontsDoctorCmd, fontsVerifyCmd, fontsReportCmd, fontsLicensesCmd)
	fontsCmd.AddCommand(unsupportedFontsCommand("vendor"), unsupportedFontsCommand("rebuild"), unsupportedFontsCommand("add"), unsupportedFontsCommand("remove"))
}
func loadFontCatalog() (*fontpacks.Catalog, error) { return fontpacks.Load("markata-fontpacks.yaml") }

func configuredFontpack() string {
	path, err := config.Discover()
	if err != nil {
		return "system"
	}
	cfg, err := config.Load(path)
	if err != nil || cfg.Fontpack == "" {
		return "system"
	}
	return cfg.Fontpack
}
func runFontsList(*cobra.Command, []string) error {
	c, err := loadFontCatalog()
	if err != nil {
		return err
	}
	for _, id := range fontpacks.SortedKeys(c.FontSources) {
		outlnf("%s\t%s", id, c.FontSources[id].Family)
	}
	return nil
}
func runFontsPacks(*cobra.Command, []string) error {
	c, err := loadFontCatalog()
	if err != nil {
		return err
	}
	for _, id := range fontpacks.SortedKeys(c.FontPacks) {
		p := c.FontPacks[id]
		outlnf("%s\t%s\t%s", id, p.Performance.Class, p.Description)
	}
	return nil
}
func runFontsShow(_ *cobra.Command, args []string) error {
	c, err := loadFontCatalog()
	if err != nil {
		return err
	}
	name := args[0]
	if p, ok := c.FontPacks[name]; ok {
		outlnf("Pack: %s\nPerformance: %s\n%s", name, p.Performance.Class, p.Description)
		return nil
	}
	if s, ok := c.FontSources[name]; ok {
		outlnf("Source: %s\nProvider: %s\nFamily: %s", name, s.Provider, s.Family)
		return nil
	}
	_, _, err = c.ResolvePack(name)
	return err
}
func runFontsDoctor(*cobra.Command, []string) error {
	py, pyErr := exec.LookPath("python")
	subset, subsetErr := exec.LookPath("pyftsubset")
	outln("FontTools doctor")
	if pyErr != nil {
		outln("  python: missing")
	} else {
		outln("  python: " + py)
	}
	if subsetErr != nil {
		outln("  pyftsubset: missing")
	} else {
		outln("  pyftsubset: " + subset)
	}
	if pyErr != nil || subsetErr != nil {
		outln("Custom font processing requires FontTools with WOFF2/Brotli support.\nRecommended:\n  uv tool install \"fonttools[woff]\"\nAlternative:\n  python -m pip install \"fonttools[woff]\"")
	}
	return nil
}
func runFontsVerify(_ *cobra.Command, args []string) error {
	c, err := loadFontCatalog()
	if err != nil {
		return err
	}
	root := c.Catalog.BundledAssetRoot
	if root == "" {
		root = "internal/fontcatalog"
	}
	lock := c.Catalog.Lockfile
	if lock == "" {
		lock = "markata-fonts.lock.yaml"
	}
	selected := ""
	if len(args) == 1 {
		selected = args[0]
	} else {
		selected = configuredFontpack()
	}
	if err := fontpacks.Verify(c, filepath.Clean(root), filepath.Clean(lock), selected); err != nil {
		return err
	}
	outln("Font catalog, manifests, licenses, hashes, and WOFF2 assets verified.")
	return nil
}
func runFontsReport(*cobra.Command, []string) error {
	c, err := loadFontCatalog()
	if err != nil {
		return err
	}
	name := "system"
	name = configuredFontpack()
	resolvedName, p, err := c.ResolvePack(name)
	if err != nil {
		return err
	}
	families := map[string]bool{}
	for _, role := range p.Roles {
		if role.Source != "" {
			families[role.Source] = true
		}
	}
	files, bytes := 0, int64(0)
	if entries, readErr := os.ReadDir(filepath.Join("output", "assets/fonts")); readErr == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, infoErr := entry.Info()
			if infoErr == nil {
				files++
				bytes += info.Size()
			}
		}
	}
	outlnf("Pack: %s\nPerformance class: %s\nFamilies: %d\nFiles emitted: %d\nTransferred font bytes: %d", resolvedName, p.Performance.Class, len(families), files, bytes)
	return nil
}
func runFontsLicenses(*cobra.Command, []string) error {
	c, err := loadFontCatalog()
	if err != nil {
		return err
	}
	outln("Licenses are recorded in family manifests and THIRD_PARTY_FONTS.md.")
	outln(strings.Join(fontpacks.SortedKeys(c.FontSources), ", "))
	return nil
}

func unsupportedFontsCommand(name string) *cobra.Command {
	return &cobra.Command{Use: name, RunE: func(*cobra.Command, []string) error {
		return fmt.Errorf("%s is reserved for the catalog maintenance workflow; use markata fonts doctor before installing FontTools", name)
	}}
}
