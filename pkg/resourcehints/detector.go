package resourcehints

import (
	"net/url"
	"regexp"
	"strings"
)

// HintType represents a resource hint type.
type HintType string

const (
	// HintTypePreconnect establishes early connections to origins.
	HintTypePreconnect HintType = "preconnect"

	// HintTypeDNSPrefetch performs DNS lookup in advance.
	HintTypeDNSPrefetch HintType = "dns-prefetch"

	// HintTypePreload fetches critical resources early.
	HintTypePreload HintType = "preload"

	// HintTypePrefetch fetches resources for future navigation.
	HintTypePrefetch HintType = "prefetch"
)

// DetectedDomain represents an external domain found in content.
type DetectedDomain struct {
	// Domain is the hostname (e.g., "fonts.googleapis.com")
	Domain string

	// Scheme is the URL scheme (e.g., "https")
	Scheme string

	// SourceType indicates where the domain was found
	SourceType string // "html", "css", "script", "font", etc.
}

// KnownDomainHint contains pre-configured hints for common services.
type KnownDomainHint struct {
	HintTypes   []HintType
	CrossOrigin string
	As          string
}

// knownDomains maps common external domains to their recommended hint types.
var knownDomains = map[string]KnownDomainHint{
	// Google Fonts - preconnect is critical for font performance
	"fonts.googleapis.com": {
		HintTypes:   []HintType{HintTypePreconnect},
		CrossOrigin: "",
	},
	"fonts.gstatic.com": {
		HintTypes:   []HintType{HintTypePreconnect},
		CrossOrigin: "anonymous",
	},

	// CDNs - dns-prefetch is usually sufficient
	"cdn.jsdelivr.net": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},
	"unpkg.com": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},
	"cdnjs.cloudflare.com": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},
	"cdn.tailwindcss.com": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},

	// Analytics - dns-prefetch to reduce impact
	"www.google-analytics.com": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},
	"www.googletagmanager.com": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},
	"analytics.google.com": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},
	"plausible.io": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},

	// Image CDNs
	"images.unsplash.com": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},
	"i.imgur.com": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},

	// Social embeds
	"platform.twitter.com": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},
	"www.youtube.com": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},
	"www.youtube-nocookie.com": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},
	"player.vimeo.com": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},
	"codepen.io": {
		HintTypes:   []HintType{HintTypeDNSPrefetch},
		CrossOrigin: "",
	},
}

// Detector detects external domains in HTML and CSS content.
type Detector struct {
	// excludeDomains is a set of domains to exclude from detection
	excludeDomains map[string]bool
}

// NewDetector creates a new Detector.
func NewDetector() *Detector {
	return &Detector{
		excludeDomains: make(map[string]bool),
	}
}

// SetExcludeDomains sets the list of domains to exclude from detection.
func (d *Detector) SetExcludeDomains(domains []string) {
	d.excludeDomains = make(map[string]bool, len(domains))
	for _, domain := range domains {
		d.excludeDomains[domain] = true
	}
}

// Regular expressions for detecting external URLs in content.
var (
	// Match URLs in href, src, srcset attributes
	hrefSrcRegex = regexp.MustCompile(`(?i)(href|src|srcset)\s*=\s*["']([^"']+)["']`)

	// Match URLs in CSS url() functions
	cssURLRegex = regexp.MustCompile(`(?i)url\s*\(\s*["']?([^"')]+)["']?\s*\)`)

	// Match URLs in inline styles
	inlineStyleRegex = regexp.MustCompile(`(?i)style\s*=\s*["'][^"']*url\s*\(\s*["']?([^"')]+)["']?\s*\)[^"']*["']`)

	// Match script src for external scripts
	scriptSrcRegex = regexp.MustCompile(`(?i)<script[^>]*\ssrc\s*=\s*["']([^"']+)["']`)

	// Match link href for external stylesheets
	linkHrefRegex = regexp.MustCompile(`(?i)<link[^>]*\shref\s*=\s*["']([^"']+)["']`)
)

// DetectExternalDomains scans HTML content and returns a list of detected external domains.
func (d *Detector) DetectExternalDomains(htmlContent string) []DetectedDomain {
	seen := make(map[string]bool)
	var domains []DetectedDomain

	// Helper to add domain if not seen
	addDomain := func(rawURL, sourceType string) {
		domain := d.extractDomain(rawURL)
		if domain == "" || seen[domain] || d.excludeDomains[domain] {
			return
		}
		seen[domain] = true
		domains = append(domains, DetectedDomain{
			Domain:     domain,
			Scheme:     d.extractScheme(rawURL),
			SourceType: sourceType,
		})
	}

	// Search for href, src, srcset attributes
	for _, match := range hrefSrcRegex.FindAllStringSubmatch(htmlContent, -1) {
		if len(match) < 3 {
			continue
		}
		attrName := strings.ToLower(match[1])
		matchedURL := match[2]
		sourceType := "html"
		if attrName == "src" {
			sourceType = "script"
		}
		addDomain(matchedURL, sourceType)
	}

	// Search for CSS url() functions
	for _, match := range cssURLRegex.FindAllStringSubmatch(htmlContent, -1) {
		if len(match) >= 2 {
			addDomain(match[1], "css")
		}
	}

	// Search for inline style URLs
	for _, match := range inlineStyleRegex.FindAllStringSubmatch(htmlContent, -1) {
		if len(match) >= 2 {
			addDomain(match[1], "style")
		}
	}

	// Search for script src specifically
	for _, match := range scriptSrcRegex.FindAllStringSubmatch(htmlContent, -1) {
		if len(match) >= 2 {
			addDomain(match[1], "script")
		}
	}

	// Search for link href specifically
	for _, match := range linkHrefRegex.FindAllStringSubmatch(htmlContent, -1) {
		if len(match) >= 2 {
			// Check if it's a stylesheet or font
			sourceType := "stylesheet"
			if strings.Contains(strings.ToLower(htmlContent), `rel="preload"`) ||
				strings.Contains(strings.ToLower(match[0]), "font") {
				sourceType = "font"
			}
			addDomain(match[1], sourceType)
		}
	}

	return domains
}

// extractDomain extracts the domain from a URL string.
// Returns empty string for relative URLs or localhost.
func (d *Detector) extractDomain(rawURL string) string {
	// Skip relative URLs
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") && !strings.HasPrefix(rawURL, "//") {
		return ""
	}

	// Handle protocol-relative URLs
	if strings.HasPrefix(rawURL, "//") {
		rawURL = "https:" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	host := parsed.Hostname()

	// Skip localhost and common local domains
	if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" ||
		strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".localhost") {
		return ""
	}

	// Skip empty hosts
	if host == "" {
		return ""
	}

	return host
}

// extractScheme extracts the URL scheme, defaulting to https.
func (d *Detector) extractScheme(rawURL string) string {
	if strings.HasPrefix(rawURL, "http://") {
		return "http"
	}
	return "https"
}

// SuggestHints suggests appropriate hints for detected domains.
// Uses known domain database for optimal hint types.
func (d *Detector) SuggestHints(domains []DetectedDomain) []SuggestedHint {
	hints := make([]SuggestedHint, 0, len(domains))

	for _, domain := range domains {
		hint := SuggestedHint{
			Domain: domain.Domain,
			Scheme: domain.Scheme,
		}

		// Check if we have known hints for this domain
		if known, ok := knownDomains[domain.Domain]; ok {
			hint.HintTypes = known.HintTypes
			hint.CrossOrigin = known.CrossOrigin
			hint.As = known.As
		} else {
			// Default to dns-prefetch for unknown domains
			hint.HintTypes = []HintType{HintTypeDNSPrefetch}
		}

		hints = append(hints, hint)
	}

	return hints
}

// SuggestedHint represents a suggested resource hint for a domain.
type SuggestedHint struct {
	Domain      string
	Scheme      string
	HintTypes   []HintType
	CrossOrigin string
	As          string
}

// GetKnownDomains returns the list of domains with predefined hints.
func GetKnownDomains() map[string]KnownDomainHint {
	// Return a copy to prevent modification
	result := make(map[string]KnownDomainHint, len(knownDomains))
	for k, v := range knownDomains {
		result[k] = v
	}
	return result
}
