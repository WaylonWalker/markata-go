// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/assets"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

const (
	webAwesomeDefaultVersion = "3.5.0"
	webAwesomeDefaultCDNBase = "https://cdn.jsdelivr.net/npm/@awesome.me/webawesome@3.5.0/dist"
	webAwesomeAssetName      = "webawesome"
	webAwesomeSourceVendor   = "vendor"
	webAwesomeComponentTag   = "tag"
)

// WebAwesomePlugin converts ergonomic markdown containers into Web Awesome components.
type WebAwesomePlugin struct {
	config webAwesomeConfig
}

type webAwesomeConfig struct {
	Enabled   bool
	Version   string
	Source    string
	CDNBase   string
	OutputDir string
	Theme     string
	Palette   string
	Brand     string
}

// NewWebAwesomePlugin creates a Web Awesome integration plugin.
func NewWebAwesomePlugin() *WebAwesomePlugin {
	return &WebAwesomePlugin{config: defaultWebAwesomeConfig()}
}

func defaultWebAwesomeConfig() webAwesomeConfig {
	return webAwesomeConfig{
		Enabled:   true,
		Version:   webAwesomeDefaultVersion,
		Source:    "vendor",
		CDNBase:   webAwesomeDefaultCDNBase,
		OutputDir: "assets/vendor/webawesome",
		Theme:     "default",
		Palette:   "default",
		Brand:     "blue",
	}
}

// Name returns the unique plugin name.
func (p *WebAwesomePlugin) Name() string {
	return "webawesome"
}

// Priority runs after markdown rendering but before templates wrap pages.
func (p *WebAwesomePlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageConfigure {
		return lifecycle.PriorityFirst
	}
	if stage == lifecycle.StageRender {
		return 50
	}
	return lifecycle.PriorityDefault
}

// Configure reads [markata-go.webawesome] from config.Extra.
func (p *WebAwesomePlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config.Extra == nil {
		return nil
	}

	webawesomeConfig, ok := config.Extra["webawesome"]
	if !ok {
		return nil
	}

	cfgMap, ok := webawesomeConfig.(map[string]interface{})
	if !ok {
		return nil
	}

	p.parseConfigMap(cfgMap)
	p.enableVendorAsset(config)

	return nil
}

func (p *WebAwesomePlugin) parseConfigMap(cfgMap map[string]interface{}) {
	if enabled, ok := cfgMap["enabled"].(bool); ok {
		p.config.Enabled = enabled
	}
	if version, ok := cfgMap["version"].(string); ok && version != "" {
		p.config.Version = version
		p.config.CDNBase = "https://cdn.jsdelivr.net/npm/@awesome.me/webawesome@" + version + "/dist"
	}
	if source, ok := cfgMap["source"].(string); ok && source != "" {
		p.config.Source = source
	}
	if cdnBase, ok := cfgMap["cdn_base_url"].(string); ok && cdnBase != "" {
		p.config.CDNBase = strings.TrimRight(cdnBase, "/")
	}
	if outputDir, ok := cfgMap["output_dir"].(string); ok && outputDir != "" {
		p.config.OutputDir = strings.Trim(outputDir, "/")
	}
	if theme, ok := cfgMap["theme"].(string); ok && theme != "" {
		p.config.Theme = theme
	}
	if palette, ok := cfgMap["palette"].(string); ok && palette != "" {
		p.config.Palette = palette
	}
	if brand, ok := cfgMap["brand"].(string); ok && brand != "" {
		p.config.Brand = brand
	}
}

// Render processes Web Awesome markdown containers and marks pages that need assets.
func (p *WebAwesomePlugin) Render(m *lifecycle.Manager) error {
	if !p.config.Enabled {
		return nil
	}

	posts := m.FilterPosts(func(post *models.Post) bool {
		if post.Skip || post.ArticleHTML == "" {
			return false
		}
		return strings.Contains(post.ArticleHTML, "webawesome") ||
			strings.Contains(post.ArticleHTML, "wa-comparison") ||
			strings.Contains(post.ArticleHTML, "<wa-")
	})

	if err := m.ProcessPostsSliceConcurrently(posts, p.processPost); err != nil {
		return err
	}

	needsWebAwesome := false
	for _, post := range m.Posts() {
		if post.ArticleHTML != "" && strings.Contains(post.ArticleHTML, "<wa-") {
			needsWebAwesome = true
			break
		}
	}

	if needsWebAwesome {
		p.enableAssets(m.Config())
	}

	return nil
}

var webAwesomeComparisonRegex = regexp.MustCompile(`(?s)<div([^>]*)class="([^"]*(?:\bwebawesome\b[^\"]*\bcomparison\b|\bwa-comparison\b)[^"]*)"([^>]*)>\s*(?:<p>\s*)?(?:<figure>\s*)?(<img\s+[^>]*>)\s*(<img\s+[^>]*>)\s*(?:</figure>\s*)?(?:</p>\s*)?</div>`)
var webAwesomeAttrRegex = regexp.MustCompile(`([A-Za-z_:][-A-Za-z0-9_:.]*)\s*=\s*("[^"]*"|'[^']*'|[^\s"'>]+)`)
var webAwesomeElementRegex = regexp.MustCompile(`<\s*(wa-[a-z0-9-]+)\b`)
var webAwesomeNestedTabsOpenRegex = regexp.MustCompile(`<div([^>]*)class="([^"]*(?:\bwebawesome\s+tabs\b|\bwa-tabs\b)[^"]*)"([^>]*)>`)
var webAwesomeNestedTabRegex = regexp.MustCompile(`(?s)<div([^>]*)class="([^"]*(?:\bwebawesome\s+tab\b|\bwa-tab\b)[^"]*)"([^>]*)>(.*?)</div>`)
var webAwesomeDivRegex = regexp.MustCompile(`(?s)<div([^>]*)class="([^"]*(?:\bwebawesome\b|\bwa-[a-z0-9-]+\b)[^"]*)"([^>]*)>(.*?)</div>`)
var webAwesomeTabMarkerRegex = regexp.MustCompile(`(?s)(?:<hr>\s*<p>tab\s+&quot;([^&]+)&quot;</p>|<p>&mdash; tab &ldquo;([^&]+)&rdquo;</p>)\s*`)
var webAwesomeFirstParagraphRegex = regexp.MustCompile(`(?s)^\s*<p>(.*?)</p>\s*`)
var webAwesomeImageRegex = regexp.MustCompile(`(?s)<img\s+[^>]*>`)
var webAwesomeCodeTextRegex = regexp.MustCompile(`(?s)<code[^>]*>(.*?)</code>`)

func (p *WebAwesomePlugin) processPost(post *models.Post) error {
	if post.Skip || post.ArticleHTML == "" {
		return nil
	}

	needsWebAwesome := strings.Contains(post.ArticleHTML, "<wa-")
	post.ArticleHTML = webAwesomeComparisonRegex.ReplaceAllStringFunc(post.ArticleHTML, func(match string) string {
		needsWebAwesome = true
		parts := webAwesomeComparisonRegex.FindStringSubmatch(match)
		if len(parts) != 6 {
			return match
		}

		attrs := parseHTMLAttrs(parts[1] + " " + parts[3])
		beforeAttrs := parseHTMLAttrs(parts[4])
		afterAttrs := parseHTMLAttrs(parts[5])
		beforeSrc := beforeAttrs["src"]
		afterSrc := afterAttrs["src"]
		if beforeSrc == "" || afterSrc == "" {
			return match
		}

		position := normalizeWebAwesomePosition(attrs["position"])
		classAttr := strings.TrimSpace("markata-webawesome-comparison " + attrs["class"])

		var b strings.Builder
		caption := attrs["caption"]
		if caption != "" {
			b.WriteString(`<figure class="markata-webawesome-figure">`)
		}

		b.WriteString(`<wa-comparison class="`)
		b.WriteString(html.EscapeString(classAttr))
		b.WriteString(`"`)
		if id := attrs["id"]; id != "" {
			b.WriteString(` id="`)
			b.WriteString(html.EscapeString(id))
			b.WriteString(`"`)
		}
		if position != "" {
			b.WriteString(` position="`)
			b.WriteString(position)
			b.WriteString(`"`)
		}
		b.WriteString(`>`)
		b.WriteString(renderWebAwesomeComparisonImage("before", beforeAttrs))
		b.WriteString(renderWebAwesomeComparisonImage("after", afterAttrs))
		b.WriteString(`</wa-comparison>`)

		if caption != "" {
			b.WriteString(`<figcaption>`)
			b.WriteString(html.EscapeString(caption))
			b.WriteString(`</figcaption></figure>`)
		}

		return b.String()
	})
	post.ArticleHTML = p.processNestedTabs(post.ArticleHTML, &needsWebAwesome)
	post.ArticleHTML = p.processGenericContainers(post.ArticleHTML, &needsWebAwesome)
	if strings.Contains(post.ArticleHTML, "<wa-") {
		needsWebAwesome = true
	}
	if needsWebAwesome {
		if post.Extra == nil {
			post.Extra = make(map[string]interface{})
		}
		post.Extra["needs_webawesome"] = true
	}

	return nil
}

func (p *WebAwesomePlugin) processNestedTabs(content string, needsWebAwesome *bool) string {
	var b strings.Builder
	for {
		loc := webAwesomeNestedTabsOpenRegex.FindStringSubmatchIndex(content)
		if loc == nil {
			b.WriteString(content)
			break
		}

		b.WriteString(content[:loc[0]])
		openTag := content[loc[0]:loc[1]]
		closeIndex := matchingDivCloseIndex(content, loc[1])
		if closeIndex < 0 {
			b.WriteString(content[loc[0]:])
			break
		}

		body := content[loc[1]:closeIndex]
		attrs := parseHTMLAttrs(openTag)
		tabMatches := webAwesomeNestedTabRegex.FindAllStringSubmatch(body, -1)
		if len(tabMatches) == 0 {
			b.WriteString(content[loc[0] : closeIndex+len("</div>")])
			content = content[closeIndex+len("</div>"):]
			continue
		}

		var tabs strings.Builder
		tabs.WriteString(`<wa-tab-group`)
		writeAllowedAttrs(&tabs, attrs)
		tabs.WriteString(`>`)
		for i, tab := range tabMatches {
			label := tabLabel(tab, i)
			panel := slugWebAwesomeTabName(label, i)
			tabs.WriteString(`<wa-tab slot="nav" panel="`)
			tabs.WriteString(panel)
			tabs.WriteString(`">`)
			tabs.WriteString(html.EscapeString(label))
			tabs.WriteString(`</wa-tab>`)
		}
		for i, tab := range tabMatches {
			label := tabLabel(tab, i)
			panel := slugWebAwesomeTabName(label, i)
			tabs.WriteString(`<wa-tab-panel name="`)
			tabs.WriteString(panel)
			tabs.WriteString(`">`)
			tabs.WriteString(strings.TrimSpace(tab[4]))
			tabs.WriteString(`</wa-tab-panel>`)
		}
		tabs.WriteString(`</wa-tab-group>`)
		*needsWebAwesome = true
		b.WriteString(tabs.String())
		content = content[closeIndex+len("</div>"):]
	}
	return b.String()
}

func matchingDivCloseIndex(content string, start int) int {
	depth := 1
	for index := start; index < len(content); {
		nextOpen := strings.Index(content[index:], "<div")
		nextClose := strings.Index(content[index:], "</div>")
		if nextClose < 0 {
			return -1
		}
		if nextOpen >= 0 && nextOpen < nextClose {
			depth++
			index += nextOpen + len("<div")
			continue
		}
		depth--
		if depth == 0 {
			return index + nextClose
		}
		index += nextClose + len("</div>")
	}
	return -1
}

func tabLabel(tab []string, index int) string {
	attrs := parseHTMLAttrs(tab[1] + " " + tab[3])
	label := attrs["label"]
	if label == "" {
		label = attrs["title"]
	}
	if label == "" {
		label = fmt.Sprintf("Tab %d", index+1)
	}
	return label
}

func (p *WebAwesomePlugin) processGenericContainers(content string, needsWebAwesome *bool) string {
	return webAwesomeDivRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := webAwesomeDivRegex.FindStringSubmatch(match)
		if len(parts) != 5 {
			return match
		}

		component := webAwesomeComponentName(strings.Fields(parts[2]))
		if component == "" || component == "comparison" {
			return match
		}

		attrs := parseHTMLAttrs(parts[1] + " " + parts[3])
		body := strings.TrimSpace(parts[4])
		rendered := renderWebAwesomeContainer(component, attrs, body)
		if rendered == "" {
			return match
		}

		*needsWebAwesome = true
		return rendered
	})
}

func webAwesomeComponentName(classes []string) string {
	for i, class := range classes {
		if class == "webawesome" && i+1 < len(classes) {
			return classes[i+1]
		}
		if strings.HasPrefix(class, "wa-") {
			return strings.TrimPrefix(class, "wa-")
		}
	}
	return ""
}

func renderWebAwesomeContainer(component string, attrs map[string]string, body string) string {
	switch component {
	case "details":
		return renderWebAwesomeDetails(attrs, body)
	case "tabs":
		return renderWebAwesomeTabs(attrs, body)
	case "copy", "copy-button":
		return renderWebAwesomeCopyButton(attrs, body)
	case "qr", "qr-code":
		return renderWebAwesomeQRCode(attrs, body)
	case "badge":
		return renderWebAwesomeSimpleComponent("badge", attrs, body)
	case webAwesomeComponentTag:
		return renderWebAwesomeSimpleComponent(webAwesomeComponentTag, attrs, body)
	case "tooltip":
		return renderWebAwesomeTooltip(attrs, body)
	case "carousel":
		return renderWebAwesomeCarousel(attrs, body)
	case "animated-image":
		return renderWebAwesomeAnimatedImage(attrs, body)
	default:
		return ""
	}
}

func renderWebAwesomeDetails(attrs map[string]string, body string) string {
	summary := attrs["summary"]
	if summary == "" {
		summary = attrs["label"]
	}
	if summary == "" {
		summary = "Details"
	}

	var b strings.Builder
	b.WriteString(`<wa-details summary="`)
	b.WriteString(html.EscapeString(summary))
	b.WriteString(`"`)
	writeAllowedAttrs(&b, attrs, "summary", "label")
	b.WriteString(`>`)
	b.WriteString(body)
	b.WriteString(`</wa-details>`)
	return b.String()
}

func renderWebAwesomeTabs(attrs map[string]string, body string) string {
	matches := webAwesomeTabMarkerRegex.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(`<wa-tab-group`)
	writeAllowedAttrs(&b, attrs)
	b.WriteString(`>`)
	for i, match := range matches {
		name := tabMatchName(body, match)
		panel := slugWebAwesomeTabName(name, i)
		b.WriteString(`<wa-tab slot="nav" panel="`)
		b.WriteString(panel)
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(name))
		b.WriteString(`</wa-tab>`)
	}
	for i, match := range matches {
		start := match[1]
		end := len(body)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		name := tabMatchName(body, match)
		panel := slugWebAwesomeTabName(name, i)
		b.WriteString(`<wa-tab-panel name="`)
		b.WriteString(panel)
		b.WriteString(`">`)
		b.WriteString(strings.TrimSpace(body[start:end]))
		b.WriteString(`</wa-tab-panel>`)
	}
	b.WriteString(`</wa-tab-group>`)
	return b.String()
}

func tabMatchName(body string, match []int) string {
	for i := 2; i+1 < len(match); i += 2 {
		if match[i] >= 0 && match[i+1] >= 0 {
			return html.UnescapeString(body[match[i]:match[i+1]])
		}
	}
	return "Tab"
}

func slugWebAwesomeTabName(name string, index int) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
		} else if b.Len() > 0 && !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = fmt.Sprintf("tab-%d", index+1)
	}
	return slug
}

func renderWebAwesomeCopyButton(attrs map[string]string, body string) string {
	value := attrs["value"]
	if value == "" {
		value = plainTextFromHTML(body)
	}
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<wa-copy-button value="`)
	b.WriteString(html.EscapeString(value))
	b.WriteString(`"`)
	writeAllowedAttrs(&b, attrs, "value")
	b.WriteString(`></wa-copy-button>`)
	return b.String()
}

func renderWebAwesomeQRCode(attrs map[string]string, body string) string {
	value := attrs["value"]
	if value == "" {
		value = plainTextFromHTML(body)
	}
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<wa-qr-code value="`)
	b.WriteString(html.EscapeString(value))
	b.WriteString(`"`)
	writeAllowedAttrs(&b, attrs, "value")
	b.WriteString(`></wa-qr-code>`)
	return b.String()
}

func renderWebAwesomeSimpleComponent(name string, attrs map[string]string, body string) string {
	body = unwrapSingleParagraph(body)
	if body == "" && attrs["label"] != "" {
		body = html.EscapeString(attrs["label"])
	}
	if body == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<wa-`)
	b.WriteString(name)
	writeAllowedAttrs(&b, attrs, "label")
	b.WriteString(`>`)
	b.WriteString(body)
	b.WriteString(`</wa-`)
	b.WriteString(name)
	b.WriteString(`>`)
	return b.String()
}

func renderWebAwesomeTooltip(attrs map[string]string, body string) string {
	content := attrs["content"]
	if content == "" {
		content = attrs["text"]
	}
	if content == "" {
		return ""
	}
	body = unwrapSingleParagraph(body)
	if body == "" && attrs["label"] != "" {
		body = html.EscapeString(attrs["label"])
	}
	if body == "" {
		return ""
	}
	// WA tooltip's default slot is the popup body; the trigger must be a
	// separate element referenced by `for=`. Render an inline anchor span
	// followed by a sibling <wa-tooltip for=...> whose default slot holds
	// the tooltip text.
	h := fnv.New64a()
	h.Write([]byte(body + "|" + content))
	sum := h.Sum(nil)
	id := "wa-tt-" + hex.EncodeToString(sum[:4])
	var b strings.Builder
	b.WriteString(`<span class="markata-wa-tooltip-anchor" id="`)
	b.WriteString(id)
	b.WriteString(`" tabindex="0">`)
	b.WriteString(body)
	b.WriteString(`</span><wa-tooltip for="`)
	b.WriteString(id)
	b.WriteString(`"`)
	writeAllowedAttrs(&b, attrs, "content", "text", "label")
	b.WriteString(`>`)
	b.WriteString(html.EscapeString(content))
	b.WriteString(`</wa-tooltip>`)
	return b.String()
}

func renderWebAwesomeCarousel(attrs map[string]string, body string) string {
	images := webAwesomeImageRegex.FindAllString(body, -1)
	if len(images) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<wa-carousel`)
	writeAllowedAttrs(&b, attrs)
	b.WriteString(`>`)
	for _, image := range images {
		b.WriteString(`<wa-carousel-item>`)
		b.WriteString(image)
		b.WriteString(`</wa-carousel-item>`)
	}
	b.WriteString(`</wa-carousel>`)
	return b.String()
}

func renderWebAwesomeAnimatedImage(attrs map[string]string, body string) string {
	image := webAwesomeImageRegex.FindString(body)
	if image == "" {
		return ""
	}
	imageAttrs := parseHTMLAttrs(image)
	if imageAttrs["src"] == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<wa-animated-image src="`)
	b.WriteString(html.EscapeString(imageAttrs["src"]))
	b.WriteString(`" alt="`)
	b.WriteString(html.EscapeString(imageAttrs["alt"]))
	b.WriteString(`"`)
	writeAllowedAttrs(&b, attrs)
	b.WriteString(`></wa-animated-image>`)
	return b.String()
}

func writeAllowedAttrs(b *strings.Builder, attrs map[string]string, skip ...string) {
	skipMap := make(map[string]bool, len(skip))
	for _, key := range skip {
		skipMap[key] = true
	}
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		if skipMap[key] || key == htmlAttrClass {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		b.WriteByte(' ')
		b.WriteString(html.EscapeString(key))
		b.WriteString(`="`)
		b.WriteString(html.EscapeString(attrs[key]))
		b.WriteString(`"`)
	}
}

func unwrapSingleParagraph(body string) string {
	matches := webAwesomeFirstParagraphRegex.FindStringSubmatch(body)
	if len(matches) == 2 && strings.TrimSpace(matches[0]) == strings.TrimSpace(body) {
		return strings.TrimSpace(matches[1])
	}
	return strings.TrimSpace(body)
}

func plainTextFromHTML(body string) string {
	if matches := webAwesomeCodeTextRegex.FindStringSubmatch(body); len(matches) == 2 {
		return strings.TrimSpace(html.UnescapeString(matches[1]))
	}
	text := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(body, "")
	return strings.TrimSpace(html.UnescapeString(text))
}

func parseHTMLAttrs(tag string) map[string]string {
	attrs := make(map[string]string)
	for _, match := range webAwesomeAttrRegex.FindAllStringSubmatch(tag, -1) {
		if len(match) != 3 {
			continue
		}
		value := strings.Trim(match[2], `"'`)
		attrs[strings.ToLower(match[1])] = html.UnescapeString(value)
	}
	return attrs
}

func normalizeWebAwesomePosition(position string) string {
	if position == "" {
		return ""
	}
	parsed, err := strconv.Atoi(position)
	if err != nil || parsed < 0 || parsed > 100 {
		return ""
	}
	return strconv.Itoa(parsed)
}

func renderWebAwesomeComparisonImage(slot string, attrs map[string]string) string {
	var b strings.Builder
	b.WriteString(`<img slot="`)
	b.WriteString(slot)
	b.WriteString(`" src="`)
	b.WriteString(html.EscapeString(attrs["src"]))
	b.WriteString(`" alt="`)
	b.WriteString(html.EscapeString(attrs["alt"]))
	b.WriteString(`" loading="lazy"`)
	if title := attrs["title"]; title != "" {
		b.WriteString(` title="`)
		b.WriteString(html.EscapeString(title))
		b.WriteString(`"`)
	}
	b.WriteString(`>`)
	return b.String()
}

func (p *WebAwesomePlugin) enableAssets(config *lifecycle.Config) {
	if config.Extra == nil {
		config.Extra = make(map[string]interface{})
	}

	cssURL := p.config.CDNBase + "/styles/webawesome.css"
	loaderURL := p.config.CDNBase + "/webawesome.loader.js"
	basePath := p.config.CDNBase
	if p.config.Source == webAwesomeSourceVendor {
		if assetURL := p.sharedAssetURL(config, webAwesomeAssetName); assetURL != "" {
			basePath = assetURL
			cssURL = basePath + "/styles/webawesome.css"
			loaderURL = basePath + "/webawesome.loader.js"
		} else {
			basePath = "/" + strings.Trim(p.config.OutputDir, "/")
			cssURL = basePath + "/styles/webawesome.css"
			loaderURL = basePath + "/webawesome.loader.js"
		}
	}

	config.Extra["webawesome_enabled"] = true
	config.Extra["webawesome_css_url"] = cssURL
	config.Extra["webawesome_loader_url"] = loaderURL
	config.Extra["webawesome_base_path"] = basePath
	config.Extra["webawesome_theme_class"] = fmt.Sprintf("wa-theme-%s wa-palette-%s wa-brand-%s", p.config.Theme, p.config.Palette, p.config.Brand)
}

func (p *WebAwesomePlugin) enableVendorAsset(config *lifecycle.Config) {
	if !p.config.Enabled || p.config.Source != webAwesomeSourceVendor {
		return
	}
	if config.Extra == nil {
		config.Extra = make(map[string]interface{})
	}

	asset := assets.Asset{
		Name:        webAwesomeAssetName,
		URL:         fmt.Sprintf("https://registry.npmjs.org/@awesome.me/webawesome/-/webawesome-%s.tgz", p.config.Version),
		LocalPath:   "webawesome",
		Integrity:   "sha512-/hJOe5vsKu9GejyTB3xFyQvvGRzXCLqdOGtBa4a+ifDNPRwzQLR3bzxcEpJsLmVfOhhem1XGbyOD9cMwefuAlA==", // pragma: allowlist secret -- npm registry SRI hash
		Version:     p.config.Version,
		Type:        "archive",
		ExtractPath: "package/dist-cdn",
	}

	existing, ok := config.Extra["cdn_assets_extra"].([]assets.Asset)
	if !ok {
		existing = nil
	}
	for _, current := range existing {
		if current.Name == asset.Name {
			return
		}
	}
	config.Extra["cdn_assets_extra"] = append(existing, asset)
}

func (p *WebAwesomePlugin) sharedAssetURL(config *lifecycle.Config, name string) string {
	if config.Extra == nil {
		return ""
	}
	if assetURLs, ok := config.Extra["asset_urls"].(map[string]string); ok {
		return assetURLs[name]
	}
	if assetURLsAny, ok := config.Extra["asset_urls"].(map[string]interface{}); ok {
		if value, ok := assetURLsAny[name].(string); ok {
			return value
		}
	}
	return ""
}

func (p *WebAwesomePlugin) componentModules(htmlContent string) []string {
	seen := make(map[string]bool)
	modules := []string{}
	for _, match := range webAwesomeElementRegex.FindAllStringSubmatch(htmlContent, -1) {
		if len(match) != 2 {
			continue
		}
		name := strings.TrimPrefix(match[1], "wa-")
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		modules = append(modules, p.componentModuleURL(name))
	}
	return modules
}

// componentModuleURL is retained for compatibility and tests; the runtime now
// uses webawesome.loader.js (autoloader) which discovers components via DOM
// observation, so per-component module URLs are not embedded in the page.
func (p *WebAwesomePlugin) componentModuleURL(name string) string {
	basePath := p.config.CDNBase
	if p.config.Source == webAwesomeSourceVendor {
		basePath = "/" + strings.Trim(p.config.OutputDir, "/")
	}
	return fmt.Sprintf("%s/components/%s/%s.js", basePath, name, name)
}

var (
	_ lifecycle.Plugin          = (*WebAwesomePlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*WebAwesomePlugin)(nil)
	_ lifecycle.RenderPlugin    = (*WebAwesomePlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*WebAwesomePlugin)(nil)
)
