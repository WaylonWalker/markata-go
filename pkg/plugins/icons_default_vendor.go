package plugins

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultLucideVendorVersion = "1.48.0"
	defaultLucideVendorTimeout = 30 * time.Second
)

// lookupIcon resolves an already-indexed icon first. If a Lucide shortcode is
// missing, the first such lookup may vendor the pinned default Lucide pack and
// re-index the local icon roots. This keeps ordinary builds network-free while
// making Lucide usable without project setup.
func (p *IconsPlugin) lookupIcon(name string) (iconAsset, bool) {
	p.iconMu.Lock()
	defer p.iconMu.Unlock()

	if asset, ok := p.icons[name]; ok {
		return asset, true
	}
	iconName, isLucide := lucideShortcodeIconName(name)
	if !isLucide || !p.autoVendor || !p.packAllowed("lucide/"+iconName) || p.vendorAttempted {
		return iconAsset{}, false
	}
	p.vendorAttempted = true

	if err := p.vendorDefaultLucide(); err != nil {
		// Icon expansion is authoring sugar, so a first-run network failure must
		// not make an otherwise valid site fail to build. Leave the shortcode
		// literal and allow an explicit [icons.vendor] configuration to enforce
		// strict vendoring when desired.
		log.Printf("[icons] default Lucide vendoring unavailable: %v", err)
		return iconAsset{}, false
	}
	if err := p.loadIcons(); err != nil {
		log.Printf("[icons] could not re-index vendored Lucide icons: %v", err)
		return iconAsset{}, false
	}
	asset, ok := p.icons[name]
	return asset, ok
}

func lucideShortcodeIconName(name string) (string, bool) {
	var iconName string
	switch {
	case strings.HasPrefix(name, "lucide/"):
		iconName = strings.TrimPrefix(name, "lucide/")
	case strings.HasPrefix(name, "lucide-"):
		iconName = strings.TrimPrefix(name, "lucide-")
	default:
		return "", false
	}
	if iconName == "" || strings.Contains(iconName, "/") {
		return "", false
	}
	return iconName, true
}

func (p *IconsPlugin) vendorDefaultLucide() error {
	vendor := NewIconVendorPlugin()
	vendor.config.enabled = true
	vendor.config.target = p.defaultVendorTarget()

	ctx, cancel := context.WithTimeout(context.Background(), defaultLucideVendorTimeout)
	defer cancel()
	return vendor.vendorPack(ctx, defaultLucideVendorPack())
}

func (p *IconsPlugin) defaultVendorTarget() string {
	for _, root := range p.paths {
		if filepath.Clean(root) == filepath.Clean(defaultIconVendorTarget) {
			return root
		}
	}
	if len(p.paths) > 0 {
		return p.paths[len(p.paths)-1]
	}
	return defaultIconVendorTarget
}

func defaultLucideVendorPack() iconVendorPack {
	return iconVendorPack{
		name:        "lucide",
		version:     defaultLucideVendorVersion,
		source:      iconVendorSourceNPM,
		packageName: "lucide-static",
		archivePath: "package",
		iconsPath:   "icons",
		licensePath: "LICENSE",
	}
}
