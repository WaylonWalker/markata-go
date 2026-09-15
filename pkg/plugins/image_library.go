package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/imageindex"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// ImageLibraryConfig controls the generated image library.
type ImageLibraryConfig = models.ImagesConfig

// ImageLibraryPlugin writes image inventory JSON artifacts and an accessible
// browser-based image library page.
type ImageLibraryPlugin struct {
	engineMu    sync.RWMutex
	engineCache map[string]*templates.Engine
}

// NewImageLibraryPlugin creates a new image library plugin.
func NewImageLibraryPlugin() *ImageLibraryPlugin {
	return &ImageLibraryPlugin{engineCache: make(map[string]*templates.Engine)}
}

// NewImagesPlugin is a compatibility constructor for the short plugin name.
func NewImagesPlugin() *ImageLibraryPlugin {
	return NewImageLibraryPlugin()
}

// Name returns the unique plugin name.
func (p *ImageLibraryPlugin) Name() string {
	return "images"
}

// Priority runs the aggregate writer after post output and before the final
// content index writer.
func (p *ImageLibraryPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageWrite {
		return lifecycle.PriorityLate
	}
	return lifecycle.PriorityDefault
}

// Write builds the canonical image index and renders the image library page.
func (p *ImageLibraryPlugin) Write(m *lifecycle.Manager) error {
	if m == nil || m.Config() == nil {
		return fmt.Errorf("image library requires a lifecycle manager and config")
	}

	config, err := parseImageLibraryConfig(m.Config().Extra)
	if err != nil {
		return err
	}
	pathPrefix, err := imageLibraryPath(config.Path)
	if err != nil {
		return err
	}
	templateName, err := imageLibraryTemplate(config.Template)
	if err != nil {
		return err
	}
	config.Template = templateName

	contentDir := m.Config().ContentDir
	if contentDir == "" {
		contentDir = "."
	}
	contentRoot, err := filepath.Abs(contentDir)
	if err != nil {
		return fmt.Errorf("resolve image library content directory: %w", err)
	}
	assetsDir := imageLibraryAssetsDir(m.Config(), contentDir)
	outputDir := m.Config().OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir
	}
	outputRoot, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("resolve image library output directory: %w", err)
	}
	pagePath := filepath.Join(outputDir, filepath.FromSlash(pathPrefix), "index.html")
	jsonPath := filepath.Join(outputDir, filepath.FromSlash(pathPrefix), "index.json")
	flatJSONPath := filepath.Join(outputDir, "images.json")
	if !config.IsEnabled() {
		return removeImageLibraryOutputs(contentRoot, GetBuildCache(m))
	}
	for _, path := range []string{pagePath, jsonPath, flatJSONPath} {
		if err := imageLibraryValidateOutputPath(outputRoot, path); err != nil {
			return err
		}
	}
	if err := imageLibraryOutputConflict(m, contentRoot, outputRoot, pagePath, jsonPath, flatJSONPath, assetsDir, pathPrefix, config.ShouldExportJSON()); err != nil {
		return err
	}
	if !config.ShouldExportJSON() {
		for _, path := range []string{jsonPath, flatJSONPath} {
			if imageLibraryOutputMatches(GetBuildCache(m), contentRoot, outputRoot, path) {
				if err := removeImageLibraryFile(outputRoot, path); err != nil {
					return fmt.Errorf("remove disabled image index export: %w", err)
				}
			}
		}
	}
	return p.writeImageLibrary(m, &config, contentDir, assetsDir, contentRoot, outputRoot, pathPrefix, pagePath, jsonPath, flatJSONPath)
}

func imageLibraryOutputConflict(m *lifecycle.Manager, contentRoot, outputRoot, pagePath, jsonPath, flatJSONPath, assetsDir, pathPrefix string, exportJSON bool) error {
	if exportJSON && imageLibraryPathsOverlap(pagePath, flatJSONPath) {
		return fmt.Errorf("image library path %q conflicts with root images.json", pathPrefix)
	}
	outputDir := m.Config().OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir
	}
	if err := imageLibraryPostOutputConflict(m, outputDir, pagePath, flatJSONPath, exportJSON); err != nil {
		return err
	}
	if err := imageLibraryFeedOutputConflict(m, outputDir, pagePath, jsonPath, flatJSONPath, exportJSON); err != nil {
		return err
	}
	if err := imageLibraryStaticOutputConflict(assetsDir, pathPrefix, exportJSON); err != nil {
		return err
	}
	cache := GetBuildCache(m)
	for _, path := range []string{pagePath, jsonPath, flatJSONPath} {
		if err := imageLibraryExistingOutputConflict(cache, contentRoot, outputRoot, path, true); err != nil {
			return err
		}
	}
	return imageLibraryStaleOutputConflict(cache, contentRoot, pagePath, jsonPath, flatJSONPath)
}

func imageLibraryPostOutputConflict(m *lifecycle.Manager, outputDir, pagePath, flatJSONPath string, exportJSON bool) error {
	for _, post := range m.Posts() {
		if post == nil || post.Skip || post.Draft {
			continue
		}
		postOutput := filepath.Join(outputDir, post.Slug, "index.html")
		if imageLibraryPathsOverlap(postOutput, pagePath) || (exportJSON && imageLibraryPathsOverlap(postOutput, flatJSONPath)) {
			return fmt.Errorf("image library output conflicts with post %q at %s", post.Path, pagePath)
		}
	}
	return nil
}

func imageLibraryFeedOutputConflict(m *lifecycle.Manager, outputDir, pagePath, jsonPath, flatJSONPath string, exportJSON bool) error {
	if cached, ok := m.Cache().Get("feed_configs"); ok {
		if feedConfigs, ok := cached.([]models.FeedConfig); ok {
			checker := NewOverwriteCheckPlugin()
			for index := range feedConfigs {
				for _, feedOutput := range checker.getFeedOutputPaths(outputDir, &feedConfigs[index]) {
					if filepath.Clean(feedOutput) == filepath.Clean(pagePath) || (exportJSON && (filepath.Clean(feedOutput) == filepath.Clean(jsonPath) || filepath.Clean(feedOutput) == filepath.Clean(flatJSONPath))) {
						return fmt.Errorf("image library output conflicts with feed %q at %s", feedConfigs[index].Slug, feedOutput)
					}
				}
			}
		}
	}
	return nil
}

func imageLibraryStaticOutputConflict(assetsDir, pathPrefix string, exportJSON bool) error {
	staticPage := filepath.Join(assetsDir, filepath.FromSlash(pathPrefix), "index.html")
	if imageLibraryPathExists(staticPage) {
		return fmt.Errorf("image library output conflicts with static file %s", staticPage)
	}
	if exportJSON {
		staticJSON := filepath.Join(assetsDir, filepath.FromSlash(pathPrefix), "index.json")
		if imageLibraryPathExists(staticJSON) {
			return fmt.Errorf("image library output conflicts with static file %s", staticJSON)
		}
		flatJSON := filepath.Join(assetsDir, "images.json")
		if imageLibraryPathExists(flatJSON) {
			return fmt.Errorf("image library output conflicts with static file %s", flatJSON)
		}
	}
	return nil
}

func imageLibraryExistingOutputConflict(cache *buildcache.Cache, contentRoot, outputRoot, path string, required bool) error {
	if !imageLibraryPathExists(path) {
		return nil
	}
	if imageLibraryOutputMatches(cache, contentRoot, outputRoot, path) {
		return nil
	}
	if imageLibraryOwnsOutput(cache, contentRoot, outputRoot, path) {
		return fmt.Errorf("image library output was modified outside the images plugin: %s", path)
	}
	if required {
		return fmt.Errorf("image library output already exists and is not owned by the images plugin: %s", path)
	}
	return nil
}

func (p *ImageLibraryPlugin) writeImageLibrary(m *lifecycle.Manager, config *ImageLibraryConfig, contentDir, assetsDir, contentRoot, outputRoot, pathPrefix, pagePath, jsonPath, flatJSONPath string) error {
	cache := GetBuildCache(m)
	inputHash, err := p.imageLibraryInputHash(m, config, contentDir, assetsDir, pathPrefix)
	if err != nil {
		return fmt.Errorf("hashing image library inputs: %w", err)
	}
	previousContentRoot := ""
	previousRoot := ""
	previousOutputs := map[string]string(nil)
	if cache != nil {
		previousContentRoot, previousRoot, previousOutputs = cache.GetImageLibraryOutputs()
	}
	if cache != nil && cache.GetImageLibraryHash() == inputHash && imageLibraryOutputsExist(pagePath, jsonPath, flatJSONPath, config.ShouldExportJSON()) {
		outputHashes, err := imageLibraryOutputHashes(pagePath, jsonPath, flatJSONPath, config.ShouldExportJSON())
		if err != nil {
			return fmt.Errorf("hash cached image library outputs: %w", err)
		}
		cache.SetImageLibraryOutputs(contentRoot, outputRoot, outputHashes)
		return nil
	}

	index, err := imageindex.Build(m.Posts(), imageindex.BuildOptions{
		ContentDir:          contentDir,
		AssetsDir:           assetsDir,
		GeneratorVersion:    imageLibraryGeneratorVersion(m.Config()),
		IncludeUnreferenced: config.ShouldIncludeUnreferenced(),
	})
	if err != nil {
		return fmt.Errorf("building image index: %w", err)
	}
	outputFS, err := openOutputRoot(outputRoot)
	if err != nil {
		return fmt.Errorf("open image library output directory: %w", err)
	}
	defer outputFS.Close()

	if config.ShouldExportJSON() {
		data, err := imageindex.Marshal(index)
		if err != nil {
			return fmt.Errorf("marshal image index: %w", err)
		}
		if err := writeImageLibraryFile(outputFS, outputRoot, jsonPath, data); err != nil {
			return fmt.Errorf("write image index: %w", err)
		}
		if err := writeImageLibraryFile(outputFS, outputRoot, flatJSONPath, data); err != nil {
			return fmt.Errorf("write root image index: %w", err)
		}
	}

	if err := p.renderPage(m, config, pathPrefix, index, outputFS, outputRoot, pagePath); err != nil {
		return err
	}
	if cache != nil {
		if err := removeStaleImageLibraryOutputs(previousContentRoot, previousRoot, previousOutputs, contentRoot, pagePath, jsonPath, flatJSONPath); err != nil {
			return fmt.Errorf("remove stale image library outputs: %w", err)
		}
		outputHashes, err := imageLibraryOutputHashes(pagePath, jsonPath, flatJSONPath, config.ShouldExportJSON())
		if err != nil {
			return fmt.Errorf("hash image library outputs: %w", err)
		}
		cache.SetImageLibraryHash(inputHash)
		cache.SetImageLibraryOutputs(contentRoot, outputRoot, outputHashes)
	}
	log.Printf("[images] Generated /%s/ with %d images", pathPrefix, len(index.Images))
	return nil
}

func parseImageLibraryConfig(extra map[string]interface{}) (ImageLibraryConfig, error) {
	config := models.NewImagesConfig()
	if extra == nil {
		return config, nil
	}
	if modelsConfig, ok := extra["models_config"].(*models.Config); ok && modelsConfig != nil {
		return normalizeImageLibraryConfig(modelsConfig.Images), nil
	}
	raw, ok := extra["images"]
	if !ok {
		if raw, ok = extra["image_library"]; !ok {
			return config, nil
		}
	}
	if typed, ok := raw.(ImageLibraryConfig); ok {
		return normalizeImageLibraryConfig(typed), nil
	}
	if typed, ok := raw.(*ImageLibraryConfig); ok && typed != nil {
		return normalizeImageLibraryConfig(*typed), nil
	}
	values, ok := raw.(map[string]interface{})
	if !ok {
		return ImageLibraryConfig{}, fmt.Errorf("images must be a table")
	}
	if value, exists := values["enabled"]; exists {
		parsed, ok := value.(bool)
		if !ok {
			return ImageLibraryConfig{}, fmt.Errorf("images.enabled must be boolean")
		}
		config.Enabled = &parsed
	}
	if value, exists := values["path"]; exists {
		parsed, ok := value.(string)
		if !ok {
			return ImageLibraryConfig{}, fmt.Errorf("images.path must be a string")
		}
		config.Path = parsed
	}
	if value, exists := values["template"]; exists {
		parsed, ok := value.(string)
		if !ok {
			return ImageLibraryConfig{}, fmt.Errorf("images.template must be a string")
		}
		config.Template = parsed
	}
	if value, exists := values["export_json"]; exists {
		parsed, ok := value.(bool)
		if !ok {
			return ImageLibraryConfig{}, fmt.Errorf("images.export_json must be boolean")
		}
		config.ExportJSON = &parsed
	}
	if value, exists := values["include_unreferenced"]; exists {
		parsed, ok := value.(bool)
		if !ok {
			return ImageLibraryConfig{}, fmt.Errorf("images.include_unreferenced must be boolean")
		}
		config.IncludeUnreferenced = &parsed
	}
	return normalizeImageLibraryConfig(config), nil
}

func normalizeImageLibraryConfig(config ImageLibraryConfig) ImageLibraryConfig {
	defaults := models.NewImagesConfig()
	if config.Enabled == nil {
		config.Enabled = defaults.Enabled
	}
	if config.Path == "" {
		config.Path = defaults.Path
	}
	if config.Template == "" {
		config.Template = defaults.Template
	}
	if config.ExportJSON == nil {
		config.ExportJSON = defaults.ExportJSON
	}
	if config.IncludeUnreferenced == nil {
		config.IncludeUnreferenced = defaults.IncludeUnreferenced
	}
	return config
}

func imageLibraryPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "images"
	}
	if filepath.IsAbs(raw) || strings.HasPrefix(raw, "/") {
		return "", fmt.Errorf("images.path must stay within output_dir: %q", raw)
	}
	clean := filepath.ToSlash(filepath.Clean(raw))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("images.path must stay within output_dir: %q", raw)
	}
	return strings.Trim(clean, "/"), nil
}

func imageLibraryTemplate(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "images.html"
	}
	if filepath.IsAbs(raw) || strings.HasPrefix(raw, "/") {
		return "", fmt.Errorf("images.template must stay within the template directory: %q", raw)
	}
	clean := filepath.ToSlash(filepath.Clean(raw))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("images.template must stay within the template directory: %q", raw)
	}
	return clean, nil
}

func imageLibraryAssetsDir(config *lifecycle.Config, contentDir string) string {
	assetsDir := ""
	if config != nil && config.Extra != nil {
		if value, ok := config.Extra["assets_dir"].(string); ok {
			assetsDir = value
		}
	}
	if assetsDir == "" {
		if configured, ok := getModelsConfig(config); ok && configured != nil {
			assetsDir = configured.AssetsDir
		}
	}
	if assetsDir == "" {
		assetsDir = "static"
	}
	if !filepath.IsAbs(assetsDir) {
		assetsDir = filepath.Join(contentDir, assetsDir)
	}
	return filepath.Clean(assetsDir)
}

func imageLibraryGeneratorVersion(config *lifecycle.Config) string {
	if config != nil && config.Extra != nil {
		if version, ok := config.Extra["markata_version"].(string); ok && version != "" {
			return version
		}
	}
	return "dev"
}

// imageLibraryRenderConfigHash fingerprints the configuration exposed to the
// image-library template. The generated page uses the full site config, not
// only the images section, so changes to site metadata, navigation, theme, or
// plugin-provided template values must invalidate the cached page.
func imageLibraryRenderConfigHash(config *lifecycle.Config) (string, error) {
	modelsConfig, ok := getModelsConfig(config)
	if !ok || modelsConfig == nil {
		modelsConfig = ToModelsConfig(config)
	}
	if modelsConfig == nil {
		return "", nil
	}

	serialized, err := json.Marshal(modelsConfig)
	if err != nil {
		return "", fmt.Errorf("marshal template configuration: %w", err)
	}
	extra, err := json.Marshal(modelsConfig.Extra)
	if err != nil {
		return "", fmt.Errorf("marshal template configuration extras: %w", err)
	}
	return buildcache.ContentHash(string(serialized) + "\x00" + string(extra)), nil
}

func (p *ImageLibraryPlugin) imageLibraryInputHash(m *lifecycle.Manager, config *ImageLibraryConfig, contentDir, assetsDir, pathPrefix string) (string, error) {
	var input strings.Builder
	input.WriteString(pathPrefix)
	input.WriteByte('\x00')
	input.WriteString(imageLibraryGeneratorVersion(m.Config()))
	input.WriteByte('\x00')
	input.WriteString(config.Template)
	input.WriteByte('\x00')
	templateHash, err := p.imageLibraryTemplateHash(m, config.Template)
	if err != nil {
		return "", err
	}
	input.WriteString(templateHash)
	input.WriteByte('\x00')
	input.WriteString(fmt.Sprintf("%t", config.IsEnabled()))
	input.WriteByte('\x00')
	input.WriteString(fmt.Sprintf("%t", config.ShouldExportJSON()))
	input.WriteByte('\x00')
	input.WriteString(fmt.Sprintf("%t", config.ShouldIncludeUnreferenced()))
	input.WriteByte('\x00')
	renderConfigHash, err := imageLibraryRenderConfigHash(m.Config())
	if err != nil {
		return "", err
	}
	input.WriteString(renderConfigHash)
	input.WriteByte('\x00')

	posts := append([]*models.Post(nil), m.Posts()...)
	sort.SliceStable(posts, func(i, j int) bool {
		leftPath, rightPath := "", ""
		leftSlug, rightSlug := "", ""
		leftHref, rightHref := "", ""
		leftContent, rightContent := "", ""
		if posts[i] != nil {
			leftPath = posts[i].Path
			leftSlug = posts[i].Slug
			leftHref = posts[i].Href
			leftContent = posts[i].Content + "\x00" + posts[i].ArticleHTML + "\x00" + posts[i].RawFrontmatter
		}
		if posts[j] != nil {
			rightPath = posts[j].Path
			rightSlug = posts[j].Slug
			rightHref = posts[j].Href
			rightContent = posts[j].Content + "\x00" + posts[j].ArticleHTML + "\x00" + posts[j].RawFrontmatter
		}
		if leftPath != rightPath {
			return leftPath < rightPath
		}
		if leftSlug != rightSlug {
			return leftSlug < rightSlug
		}
		if leftHref != rightHref {
			return leftHref < rightHref
		}
		return leftContent < rightContent
	})
	for _, post := range posts {
		if post == nil || !post.Published || post.Draft || post.Private || post.Skip {
			continue
		}
		input.WriteString(post.Path)
		input.WriteByte('\x00')
		input.WriteString(post.Slug)
		input.WriteByte('\x00')
		input.WriteString(post.Href)
		input.WriteByte('\x00')
		input.WriteString(post.PlainTitle())
		input.WriteByte('\x00')
		if post.Date != nil {
			input.WriteString(post.Date.UTC().Format(time.RFC3339Nano))
		}
		input.WriteByte('\x00')
		input.WriteString(buildcache.ContentHash(strings.Join([]string{
			post.InputHash,
			post.Content,
			post.ArticleHTML,
			post.RawFrontmatter,
			frontmatterImageHash(post),
		}, "\x00")))
		input.WriteByte('\x00')
	}

	assetHash, assetStateHash, err := hashImageDirectory(assetsDir, imageHashExtensions())
	if err != nil {
		return "", err
	}
	// Referenced local images may live beside content instead of below the
	// configured assets directory. Include the content tree so changes to
	// their dimensions, timestamps, or bytes invalidate the generated index.
	contentImageHash, contentImageStateHash, err := hashImageDirectory(contentDir, imageHashExtensions())
	if err != nil {
		return "", err
	}
	input.WriteString(assetHash)
	input.WriteByte('\x00')
	input.WriteString(assetStateHash)
	input.WriteByte('\x00')
	input.WriteString(contentImageHash)
	input.WriteByte('\x00')
	input.WriteString(contentImageStateHash)
	return buildcache.ContentHash(input.String()), nil
}

func (p *ImageLibraryPlugin) imageLibraryTemplateHash(m *lifecycle.Manager, templateName string) (string, error) {
	var input strings.Builder
	input.WriteString(imageLibraryGeneratorVersion(m.Config()))
	input.WriteByte('\x00')
	if cache := GetBuildCache(m); cache != nil {
		input.WriteString(cache.GetTemplatesHash())
		input.WriteByte('\x00')
	}

	engine, err := p.templateEngine(m.Config())
	if err != nil {
		return "", err
	}
	paths := engine.SearchPaths()
	for _, path := range paths {
		hash, err := buildcache.HashDirectory(path, []string{".html", ".txt", ".md"})
		if err != nil {
			return "", err
		}
		input.WriteString(filepath.Clean(path))
		input.WriteByte('=')
		input.WriteString(hash)
		input.WriteByte('\x00')
	}
	if len(paths) == 0 && engine.HasEmbeddedTemplate(templateName) {
		input.WriteString("embedded:")
		input.WriteString(templateName)
	}
	return buildcache.ContentHash(input.String()), nil
}

func frontmatterImageHash(post *models.Post) string {
	if post == nil || post.Extra == nil {
		return ""
	}
	keys := make([]string, 0, len(post.Extra))
	for key := range post.Extra {
		for _, field := range []string{"image", "video", "cover", "cover_image", "og_image", "social_image", "thumbnail", "featured_image", "hero_image", "avatar", "author_image", "poster_image", "poster", "video_poster", "video_thumbnail", "thumb", "image_alt", "video_alt", "cover_alt", "cover_image_alt", "alt"} {
			if key == field {
				keys = append(keys, key)
				break
			}
		}
	}
	sort.Strings(keys)
	var result strings.Builder
	for _, key := range keys {
		result.WriteString(key)
		result.WriteByte('=')
		result.WriteString(fmt.Sprint(post.Extra[key]))
		result.WriteByte('\x00')
	}
	return buildcache.ContentHash(result.String())
}

func imageHashExtensions() []string {
	return []string{
		".avif", ".avi", ".bmp", ".gif", ".heic", ".heif", ".ico", ".jpeg", ".jpg", ".m4v", ".mkv", ".mov", ".mp4", ".ogv", ".ogg", ".png", ".svg", ".tif", ".tiff", ".webm", ".webp",
		".AVIF", ".AVI", ".BMP", ".GIF", ".HEIC", ".HEIF", ".ICO", ".JPEG", ".JPG", ".M4V", ".MKV", ".MOV", ".MP4", ".OGV", ".OGG", ".PNG", ".SVG", ".TIF", ".TIFF", ".WEBM", ".WEBP",
	}
}

// hashImageDirectory fingerprints regular image and video files while ignoring
// symbolic links. Media discovery intentionally skips symlinks, so the cache
// hash must not fail on a dangling or unreadable media link in content or assets.
func hashImageDirectory(dir string, extensions []string) (contentHash, stateHash string, err error) {
	extensionsByName := make(map[string]struct{}, len(extensions))
	for _, extension := range extensions {
		extensionsByName[extension] = struct{}{}
	}
	type imageFile struct {
		path     string
		fullPath string
		size     int64
		modTime  int64
	}
	files := make([]imageFile, 0, 32)
	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			return nil
		}
		if _, ok := extensionsByName[strings.ToLower(filepath.Ext(path))]; !ok {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		relative, relativeErr := filepath.Rel(dir, path)
		if relativeErr != nil {
			relative = path
		}
		files = append(files, imageFile{
			path:     relative,
			fullPath: path,
			size:     info.Size(),
			modTime:  info.ModTime().UnixNano(),
		})
		return nil
	})
	if os.IsNotExist(err) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })

	contentHasher := sha256.New()
	stateHasher := sha256.New()
	for _, file := range files {
		contentHasher.Write([]byte(file.path))
		media, err := os.Open(file.fullPath)
		if err != nil {
			return "", "", err
		}
		if _, err := io.Copy(contentHasher, media); err != nil {
			_ = media.Close()
			return "", "", fmt.Errorf("hash media file %s: %w", file.fullPath, err)
		}
		if err := media.Close(); err != nil {
			return "", "", fmt.Errorf("close media file %s: %w", file.fullPath, err)
		}
		stateHasher.Write([]byte(file.path))
		stateHasher.Write([]byte{0})
		stateHasher.Write([]byte(strconv.FormatInt(file.size, 10)))
		stateHasher.Write([]byte{0})
		stateHasher.Write([]byte(strconv.FormatInt(file.modTime, 10)))
		stateHasher.Write([]byte{0})
	}
	return hex.EncodeToString(contentHasher.Sum(nil)), hex.EncodeToString(stateHasher.Sum(nil)), nil
}

func imageLibraryOutputsExist(pagePath, jsonPath, flatJSONPath string, exportJSON bool) bool {
	if !isRegularImageLibraryFile(pagePath) {
		return false
	}
	if exportJSON {
		if !isRegularImageLibraryFile(jsonPath) || !isRegularImageLibraryFile(flatJSONPath) {
			return false
		}
	}
	return true
}

func isRegularImageLibraryFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func imageLibraryPathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func imageLibraryPathsOverlap(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if left == right {
		return true
	}
	for _, pair := range [][2]string{{left, right}, {right, left}} {
		relative, err := filepath.Rel(pair[1], pair[0])
		if err != nil || relative == "." || relative == ".." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		return true
	}
	return false
}

func imageLibraryOwnsOutput(cache *buildcache.Cache, contentRoot, outputRoot, path string) bool {
	return cache != nil && cache.OwnsImageLibraryOutput(contentRoot, outputRoot, path)
}

func imageLibraryOutputMatches(cache *buildcache.Cache, contentRoot, outputRoot, path string) bool {
	if !imageLibraryOwnsOutput(cache, contentRoot, outputRoot, path) || !isRegularImageLibraryFile(path) {
		return false
	}
	expected, ok := cache.ImageLibraryOutputHash(contentRoot, outputRoot, path)
	if !ok || expected == "" {
		return false
	}
	actual, err := buildcache.HashFile(path)
	return err == nil && actual == expected
}

func imageLibraryOutputHashes(pagePath, jsonPath, flatJSONPath string, exportJSON bool) (map[string]string, error) {
	paths := []string{pagePath}
	if exportJSON {
		paths = append(paths, jsonPath, flatJSONPath)
	}
	hashes := make(map[string]string, len(paths))
	for _, path := range paths {
		if !isRegularImageLibraryFile(path) {
			continue
		}
		hash, err := buildcache.HashFile(path)
		if err != nil {
			return nil, err
		}
		hashes[path] = hash
	}
	return hashes, nil
}

func imageLibraryStaleOutputConflict(cache *buildcache.Cache, contentRoot, pagePath, jsonPath, flatJSONPath string) error {
	if cache == nil {
		return nil
	}
	previousContentRoot, previousRoot, outputHashes := cache.GetImageLibraryOutputs()
	activeContentRoot, err := filepath.Abs(contentRoot)
	if err != nil || filepath.Clean(previousContentRoot) != filepath.Clean(activeContentRoot) {
		return nil
	}
	if previousRoot == "" {
		return nil
	}
	previousRoot, err = filepath.Abs(previousRoot)
	if err != nil {
		return nil
	}
	currentPage, err := filepath.Abs(pagePath)
	if err != nil {
		return err
	}
	currentJSON, err := filepath.Abs(jsonPath)
	if err != nil {
		return err
	}
	currentFlatJSON, err := filepath.Abs(flatJSONPath)
	if err != nil {
		return err
	}
	current := map[string]struct{}{
		filepath.Clean(currentPage):     {},
		filepath.Clean(currentJSON):     {},
		filepath.Clean(currentFlatJSON): {},
	}
	for relative, expected := range outputHashes {
		path, err := imageLibraryOutputPath(previousRoot, relative)
		if err != nil {
			return fmt.Errorf("invalid cached image library output %q: %w", relative, err)
		}
		if _, ok := current[filepath.Clean(path)]; ok || !imageLibraryPathExists(path) {
			continue
		}
		if !isRegularImageLibraryFile(path) {
			return fmt.Errorf("image library output is no longer a regular file: %s", path)
		}
		actual, err := buildcache.HashFile(path)
		if err != nil {
			return fmt.Errorf("hash image library output %s: %w", path, err)
		}
		if actual != expected {
			return fmt.Errorf("image library output was modified outside the images plugin: %s", path)
		}
	}
	return nil
}

func (p *ImageLibraryPlugin) renderPage(m *lifecycle.Manager, config *ImageLibraryConfig, pathPrefix string, index imageindex.Index, outputFS *os.Root, outputRoot, pagePath string) error {
	engine, err := p.templateEngine(m.Config())
	if err != nil {
		return err
	}
	engine.ClearCache()
	// Config maps are cached by pointer. Clear the cache because configuration
	// can be updated in place between incremental writes.
	templates.ClearConfigMapCache()
	if !engine.TemplateExists(config.Template) {
		log.Printf("[images] Warning: template %q not found, skipping image library page", config.Template)
		if err := removeImageLibraryFileFromRoot(outputFS, outputRoot, pagePath); err != nil {
			return fmt.Errorf("remove image library page: %w", err)
		}
		return nil
	}

	page := newImageLibraryPage(index)
	if config.ShouldExportJSON() {
		page.JSONHref = "/" + strings.Trim(pathPrefix, "/") + "/index.json"
	}
	title := "Image Library"
	description := "Browse the images and videos used by this site."
	syntheticPost := &models.Post{Slug: pathPrefix, Href: "/" + pathPrefix + "/", Published: true, Extra: make(map[string]interface{})}
	syntheticPost.Title = &title
	syntheticPost.Description = &description
	modelsConfig, ok := getModelsConfig(m.Config())
	if !ok || modelsConfig == nil {
		modelsConfig = ToModelsConfig(m.Config())
	}
	if modelsConfig == nil {
		modelsConfig = &models.Config{}
	}
	ctx := templates.NewContext(syntheticPost, "", modelsConfig)
	ctx.Extra["title"] = title
	ctx.Extra["description"] = description
	ctx.Extra["image_library"] = page
	ctx.Extra["image_index"] = index
	ctx.Extra["image_library_config"] = *config
	ctx.Extra["needs_image_zoom"] = false

	html, err := engine.Render(config.Template, ctx)
	if err != nil {
		return fmt.Errorf("rendering image library template: %w", err)
	}
	if err := writeImageLibraryFile(outputFS, outputRoot, pagePath, []byte(html)); err != nil {
		return fmt.Errorf("writing image library page: %w", err)
	}
	return nil
}

func (p *ImageLibraryPlugin) templateEngine(config *lifecycle.Config) (*templates.Engine, error) {
	if config != nil && config.Extra != nil {
		if value, ok := config.Extra["templates.engine"]; ok {
			if engine, ok := value.(*templates.Engine); ok && engine != nil {
				return engine, nil
			}
		}
	}
	templatesDir := PluginNameTemplates
	if config != nil && config.Extra != nil {
		if value, ok := config.Extra["templates_dir"].(string); ok && value != "" {
			templatesDir = value
		}
	}
	themeName := getThemeName(config)
	cacheKey := templatesDir + ":" + themeName
	p.engineMu.RLock()
	engine := p.engineCache[cacheKey]
	p.engineMu.RUnlock()
	if engine != nil {
		return engine, nil
	}
	p.engineMu.Lock()
	defer p.engineMu.Unlock()
	if cached := p.engineCache[cacheKey]; cached != nil {
		return cached, nil
	}
	var err error
	engine, err = templates.NewEngineWithTheme(templatesDir, themeName)
	if err != nil {
		return nil, err
	}
	p.engineCache[cacheKey] = engine
	return engine, nil
}

// ImageLibraryPage is the template view model for the generated page.
type ImageLibraryPage struct {
	Images      []ImageCard
	Count       int
	UsedCount   int
	UnusedCount int
	CoverCount  int
	JSONHref    string
}

// ImageCard contains the presentation-only values derived from one image.
type ImageCard struct {
	Src            string
	PreviewSrc     string
	PosterSrc      string
	Srcset         string
	Name           string
	Alt            string
	MIMEType       string
	Width          int
	Height         int
	AspectRatio    string
	AddedAt        string
	AddedAtUnix    int64
	LastUsedAt     string
	LastUsedAtUnix int64
	Markdown       string
	SearchText     string
	Cover          bool
	Embed          bool
	IsVideo        bool
	Used           bool
	Uses           []imageindex.Use
	MoreUses       []imageindex.Use
	MoreCount      int
	UseCount       int
}

func newImageLibraryPage(index imageindex.Index) ImageLibraryPage {
	page := ImageLibraryPage{Images: make([]ImageCard, 0, len(index.Images)), Count: len(index.Images)}
	for i := range index.Images {
		image := &index.Images[i]
		isVideo := imageIsVideo(*image)
		srcset := ""
		if !isVideo {
			srcset = imageSrcset(image.Src)
		}
		card := ImageCard{
			Src:         image.Src,
			PreviewSrc:  imagePreviewURL(image.Src),
			PosterSrc:   imagePosterURL(*image, isVideo),
			Srcset:      srcset,
			Name:        imageName(image.Src),
			Alt:         image.Alt,
			MIMEType:    image.MIMEType,
			Width:       image.Width,
			Height:      image.Height,
			AspectRatio: imageAspectRatio(image.Width, image.Height),
			Markdown:    markdownForImage(image.Src, image.Alt),
			SearchText:  imageSearchText(*image),
			Cover:       image.Cover,
			Embed:       image.Embed,
			IsVideo:     isVideo,
			Used:        len(image.Uses) > 0,
			UseCount:    len(image.Uses),
		}
		if image.AddedAt != nil {
			card.AddedAt = image.AddedAt.UTC().Format("Jan 2, 2006")
			card.AddedAtUnix = image.AddedAt.Unix()
		}
		if image.LastUsedAt != nil {
			card.LastUsedAt = image.LastUsedAt.UTC().Format("Jan 2, 2006")
			card.LastUsedAtUnix = image.LastUsedAt.Unix()
		}
		if len(image.Uses) > 3 {
			card.Uses = append([]imageindex.Use(nil), image.Uses[:3]...)
			card.MoreUses = append([]imageindex.Use(nil), image.Uses[3:]...)
			card.MoreCount = len(card.MoreUses)
		} else {
			card.Uses = append([]imageindex.Use(nil), image.Uses...)
		}
		page.Images = append(page.Images, card)
		if card.Used {
			page.UsedCount++
		} else {
			page.UnusedCount++
		}
		if card.Cover {
			page.CoverCount++
		}
	}
	return page
}

func imagePreviewURL(src string) string {
	parsed, err := url.Parse(src)
	if err != nil || parsed.Host == "" || !templates.IsTrustedMediaURL(src) {
		return src
	}
	return templates.WithSize(src, 640, 0)
}

func imageIsVideo(image imageindex.Image) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(image.MIMEType)), "video/") || templates.IsVideoURL(image.Src)
}

func imagePosterURL(image imageindex.Image, isVideo bool) string {
	if !isVideo {
		return ""
	}
	poster := strings.TrimSpace(image.PosterSrc)
	if poster == "" {
		poster = templates.PosterURLFromMap(map[string]interface{}{}, image.Src)
	}
	if poster == "" {
		return ""
	}
	return templates.WithSize(poster, 640, 0)
}

func imageSrcset(src string) string {
	if templates.IsVideoURL(src) {
		return ""
	}
	parsed, err := url.Parse(src)
	if err != nil || parsed.Host == "" || !templates.IsTrustedMediaURL(src) {
		return ""
	}
	widths := []int{320, 640, 960, 1280}
	values := make([]string, 0, len(widths))
	for _, width := range widths {
		values = append(values, fmt.Sprintf("%s %dw", templates.WithSize(src, width, 0), width))
	}
	return strings.Join(values, ", ")
}

func imageName(src string) string {
	parsed, err := url.Parse(src)
	if err == nil && parsed.Path != "" {
		name := filepath.Base(parsed.Path)
		if name != "." && name != string(filepath.Separator) && name != "" {
			return name
		}
	}
	return src
}

func imageAspectRatio(width, height int) string {
	if width > 0 && height > 0 {
		return fmt.Sprintf("%d / %d", width, height)
	}
	return "4 / 3"
}

func markdownForImage(src, alt string) string {
	alt = strings.ReplaceAll(strings.ReplaceAll(alt, `\`, `\\`), "]", `\]`)
	src = strings.ReplaceAll(strings.ReplaceAll(src, `\`, `\\`), ")", `\)`)
	return fmt.Sprintf("![%s](%s)", alt, src)
}

func imageSearchText(image imageindex.Image) string {
	values := []string{image.Src, image.Alt}
	if imageIsVideo(image) {
		values = append(values, "video")
	}
	if image.Embed {
		values = append(values, "embed")
	}
	for _, use := range image.Uses {
		values = append(values, use.Post, use.Href, use.Title, use.Caption)
	}
	return strings.ToLower(strings.Join(values, " "))
}

var imageLibraryTempCounter uint64

func writeImageLibraryFile(outputFS *os.Root, outputRoot, path string, data []byte) error {
	relative, err := outputRelativePath(outputRoot, path)
	if err != nil {
		return err
	}
	directory := filepath.Dir(relative)
	if directory != "." {
		if err := outputFS.MkdirAll(directory, 0o755); err != nil {
			return err
		}
	}
	temporary, temporaryName, err := createImageLibraryTemp(outputFS, directory)
	if err != nil {
		return err
	}
	defer outputFS.Remove(temporaryName) //nolint:errcheck // best-effort cleanup after rename or failure
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if runtime.GOOS == tailwindOSWindows {
		if err := removeImageLibraryFileFromRoot(outputFS, outputRoot, path); err != nil {
			return err
		}
	}
	return outputFS.Rename(temporaryName, relative)
}

func createImageLibraryTemp(outputFS *os.Root, directory string) (*os.File, string, error) {
	for attempt := 0; attempt < 100; attempt++ {
		name := fmt.Sprintf(".images-%d-%d.tmp", os.Getpid(), atomic.AddUint64(&imageLibraryTempCounter, 1))
		if directory != "." {
			name = filepath.Join(directory, name)
		}
		file, err := outputFS.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return file, name, nil
		}
		if !os.IsExist(err) {
			return nil, "", err
		}
	}
	return nil, "", fmt.Errorf("create temporary image library output: too many existing temporary files")
}

func removeImageLibraryFile(outputRoot, path string) error {
	outputFS, err := openExistingOutputRoot(outputRoot)
	if err != nil || outputFS == nil {
		return err
	}
	defer outputFS.Close()
	return removeImageLibraryFileFromRoot(outputFS, outputRoot, path)
}

func removeImageLibraryFileFromRoot(outputFS *os.Root, outputRoot, path string) error {
	relative, err := outputRelativePath(outputRoot, path)
	if err != nil {
		return err
	}
	if err := outputFS.Remove(relative); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func removeImageLibraryOutputs(contentRoot string, cache *buildcache.Cache) error {
	if cache == nil {
		return nil
	}
	previousContentRoot, previousRoot, outputHashes := cache.GetImageLibraryOutputs()
	activeContentRoot, err := filepath.Abs(contentRoot)
	if err != nil {
		return fmt.Errorf("resolve image library content directory: %w", err)
	}
	if filepath.Clean(previousContentRoot) != filepath.Clean(activeContentRoot) {
		cache.ClearImageLibraryOutputs()
		cache.SetImageLibraryHash("")
		return nil
	}
	if previousRoot == "" {
		cache.ClearImageLibraryOutputs()
		cache.SetImageLibraryHash("")
		return nil
	}
	previousRoot, err = filepath.Abs(previousRoot)
	if err != nil {
		return fmt.Errorf("resolve cached image library output directory: %w", err)
	}
	outputFS, err := openExistingOutputRoot(previousRoot)
	if err != nil {
		return fmt.Errorf("open cached image library output directory: %w", err)
	}
	if outputFS == nil {
		cache.ClearImageLibraryOutputs()
		cache.SetImageLibraryHash("")
		return nil
	}
	defer outputFS.Close()
	for relative, expected := range outputHashes {
		path, err := imageLibraryOutputPath(previousRoot, relative)
		if err != nil {
			return fmt.Errorf("invalid cached image library output %q: %w", relative, err)
		}
		if !imageLibraryPathExists(path) {
			continue
		}
		if !isRegularImageLibraryFile(path) {
			return fmt.Errorf("image library output is no longer a regular file: %s", path)
		}
		actual, err := buildcache.HashFile(path)
		if err != nil {
			return fmt.Errorf("hash disabled image library output %s: %w", path, err)
		}
		if actual != expected {
			return fmt.Errorf("image library output was modified outside the images plugin: %s", path)
		}
		if err := removeImageLibraryFileFromRoot(outputFS, previousRoot, path); err != nil {
			return fmt.Errorf("remove disabled image library output %s: %w", path, err)
		}
	}
	cache.ClearImageLibraryOutputs()
	cache.SetImageLibraryHash("")
	return nil
}

func imageLibraryOutputPath(outputRoot, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) {
		return "", fmt.Errorf("path must be relative to output root")
	}
	cleaned := filepath.Clean(relative)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes output root")
	}
	path := filepath.Join(outputRoot, cleaned)
	if err := imageLibraryValidateOutputPath(outputRoot, path); err != nil {
		return "", err
	}
	return path, nil
}

func imageLibraryValidateOutputPath(outputRoot, path string) error {
	return validateOutputPath(outputRoot, path)
}

func removeStaleImageLibraryOutputs(previousContentRoot, previousRoot string, previous map[string]string, contentRoot, pagePath, jsonPath, flatJSONPath string) error {
	activeContentRoot, err := filepath.Abs(contentRoot)
	if err != nil {
		return err
	}
	if filepath.Clean(previousContentRoot) != filepath.Clean(activeContentRoot) {
		return nil
	}
	if previousRoot == "" {
		return nil
	}
	previousRoot, err = filepath.Abs(previousRoot)
	if err != nil {
		return err
	}
	outputFS, err := openExistingOutputRoot(previousRoot)
	if err != nil {
		return fmt.Errorf("open cached image library output directory: %w", err)
	}
	if outputFS == nil {
		return nil
	}
	defer outputFS.Close()
	currentPage, err := filepath.Abs(pagePath)
	if err != nil {
		return err
	}
	currentJSON, err := filepath.Abs(jsonPath)
	if err != nil {
		return err
	}
	currentFlatJSON, err := filepath.Abs(flatJSONPath)
	if err != nil {
		return err
	}
	current := map[string]struct{}{
		filepath.Clean(currentPage):     {},
		filepath.Clean(currentJSON):     {},
		filepath.Clean(currentFlatJSON): {},
	}
	for relative, expected := range previous {
		path, err := imageLibraryOutputPath(previousRoot, relative)
		if err != nil {
			return err
		}
		if _, ok := current[path]; ok {
			continue
		}
		if !imageLibraryPathExists(path) {
			continue
		}
		if !isRegularImageLibraryFile(path) {
			return fmt.Errorf("image library output is no longer a regular file: %s", path)
		}
		actual, err := buildcache.HashFile(path)
		if err != nil {
			return err
		}
		if actual != expected {
			return fmt.Errorf("image library output was modified outside the images plugin: %s", path)
		}
		if err := removeImageLibraryFileFromRoot(outputFS, previousRoot, path); err != nil {
			return err
		}
	}
	return nil
}

// Ensure ImageLibraryPlugin implements the lifecycle contracts.
var (
	_ lifecycle.Plugin         = (*ImageLibraryPlugin)(nil)
	_ lifecycle.WritePlugin    = (*ImageLibraryPlugin)(nil)
	_ lifecycle.PriorityPlugin = (*ImageLibraryPlugin)(nil)
)
