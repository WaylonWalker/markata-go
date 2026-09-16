package imageindex

import (
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"image"
	_ "image/gif"  // Register GIF decoding for local metadata.
	_ "image/jpeg" // Register JPEG decoding for local metadata.
	_ "image/png"  // Register PNG decoding for local metadata.
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	figure "github.com/mangoumbrella/goldmark-figure"
	figureast "github.com/mangoumbrella/goldmark-figure/ast"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
	"golang.org/x/net/html"

	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

// BuildOptions controls image discovery.
type BuildOptions struct {
	ContentDir          string
	AssetsDir           string
	GeneratorVersion    string
	IncludeUnreferenced bool
}

// Build discovers public media references and local asset files.
func Build(posts []*models.Post, options BuildOptions) (Index, error) {
	contentDir := options.ContentDir
	if contentDir == "" {
		contentDir = "."
	}
	if absolute, err := filepath.Abs(contentDir); err == nil {
		contentDir = filepath.Clean(absolute)
	}
	assetsDir := options.AssetsDir
	if assetsDir == "" {
		assetsDir = filepath.Join(contentDir, "static")
	} else if !filepath.IsAbs(assetsDir) {
		assetsDir = filepath.Join(contentDir, assetsDir)
	}
	if absolute, err := filepath.Abs(assetsDir); err == nil {
		assetsDir = filepath.Clean(absolute)
	}

	catalog := newCatalog(contentDir, assetsDir)
	orderedPosts := imagePostsInStableOrder(posts)
	if options.IncludeUnreferenced {
		if err := catalog.scanAssets(); err != nil {
			return Index{}, err
		}
	}

	for _, post := range orderedPosts {
		if post != nil && post.Private {
			catalog.markPrivatePost(post)
		}
	}
	for _, post := range orderedPosts {
		if !isPublicPost(post) {
			continue
		}
		for _, reference := range extractMarkdownImages(post.Content) {
			catalog.addReference(reference, post, false)
		}
		for _, reference := range extractHTMLMedia(post.ArticleHTML) {
			catalog.addReference(reference, post, false)
		}
		// This fallback covers direct callers that have not run Render yet and
		// raw HTML or video elements that Goldmark represents as HTML nodes.
		for _, reference := range extractHTMLMedia(post.Content) {
			catalog.addReference(reference, post, false)
		}
		catalog.addFrontmatterImages(post)
	}
	catalog.removePrivateOnlyImages()

	version := options.GeneratorVersion
	if version == "" {
		version = "dev"
	}
	index := catalog.index(Generator{Name: GeneratorName, Version: version})
	return normalize(index)
}

type imageReference struct {
	src      string
	alt      string
	caption  string
	poster   string
	mimeType string
	embed    bool
	video    bool
}

type catalog struct {
	contentDir    string
	assetsDir     string
	byKey         map[string]*catalogImage
	byLocal       map[string]*catalogImage
	privateLocals map[string]struct{}
}

type catalogImage struct {
	image      Image
	local      string
	altRank    int
	posterRank int
	uses       map[string]Use
}

func newCatalog(contentDir, assetsDir string) *catalog {
	return &catalog{
		contentDir:    contentDir,
		assetsDir:     assetsDir,
		byKey:         make(map[string]*catalogImage),
		byLocal:       make(map[string]*catalogImage),
		privateLocals: make(map[string]struct{}),
	}
}

func (c *catalog) scanAssets() error {
	err := filepath.WalkDir(c.assetsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() || !isMediaPath(path) {
			return nil
		}

		absolute, err := filepath.Abs(path)
		if err != nil {
			absolute = filepath.Clean(path)
		}
		relative, err := filepath.Rel(c.assetsDir, path)
		if err != nil {
			return err
		}
		src := "/" + filepath.ToSlash(relative)
		imageRecord := c.ensure("local:"+absolute, c.localSource(absolute, src))
		imageRecord.local = absolute
		c.byLocal[absolute] = imageRecord
		c.populateLocalMetadata(imageRecord, path)
		return nil
	})
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *catalog) addReference(reference imageReference, post *models.Post, cover bool) {
	rawSrc := strings.TrimSpace(reference.src)
	if rawSrc == "" || ignoredSource(rawSrc) {
		return
	}

	localPath := c.resolveLocalPath(rawSrc, post)
	var imageRecord *catalogImage
	if localPath != "" {
		imageRecord = c.byLocal[localPath]
		if imageRecord == nil {
			imageRecord = c.ensure("local:"+localPath, c.localSource(localPath, rawSrc))
			imageRecord.local = localPath
			c.byLocal[localPath] = imageRecord
			c.populateLocalMetadata(imageRecord, localPath)
		}
	} else {
		if c.rejectedLocalSource(rawSrc, post) {
			return
		}
		imageRecord = c.ensure("url:"+rawSrc, rawSrc)
		if width, height, ok := templates.MediaDimensionsFromURL(rawSrc); ok {
			if imageRecord.image.Width == 0 {
				imageRecord.image.Width = width
			}
			if imageRecord.image.Height == 0 {
				imageRecord.image.Height = height
			}
		}
	}

	if imageRecord.image.MIMEType == "" {
		imageRecord.image.MIMEType = strings.TrimSpace(reference.mimeType)
	}
	if imageRecord.image.MIMEType == "" {
		imageRecord.image.MIMEType = mimeType(rawSrc)
	}
	if reference.video || isVideoSource(rawSrc, reference.mimeType) || strings.HasPrefix(strings.ToLower(imageRecord.image.MIMEType), "video/") {
		if poster, rank := videoPoster(reference, post, rawSrc); poster != "" && rank > imageRecord.posterRank {
			imageRecord.image.PosterSrc = poster
			imageRecord.posterRank = rank
		}
	}

	if reference.alt == "" {
		c.setAlt(imageRecord, fallbackAlt(rawSrc), 1)
	} else {
		c.setAlt(imageRecord, reference.alt, 2)
	}
	c.addUse(imageRecord, post, cover, reference.embed, reference.caption)
}

func isVideoSource(rawSrc, declaredMIME string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(declaredMIME)), "video/") || templates.IsVideoURL(rawSrc)
}

func videoPoster(reference imageReference, post *models.Post, mediaURL string) (poster string, rank int) {
	if post != nil && post.Extra != nil {
		if poster := templates.PosterURLFromMap(post.Extra, ""); poster != "" {
			return poster, 3
		}
	}
	if strings.TrimSpace(reference.poster) != "" {
		return templates.PosterURLFromMap(map[string]interface{}{"poster": reference.poster}, ""), 2
	}
	return templates.PosterURLFromMap(map[string]interface{}{}, mediaURL), 1
}

func (c *catalog) markPrivatePost(post *models.Post) {
	for _, reference := range extractMarkdownImages(post.Content) {
		c.markPrivateSource(reference.src, post)
	}
	for _, reference := range extractHTMLMedia(post.ArticleHTML) {
		c.markPrivateSource(reference.src, post)
	}
	for _, reference := range extractHTMLMedia(post.Content) {
		c.markPrivateSource(reference.src, post)
	}
	if post.Extra == nil {
		return
	}
	for _, field := range frontmatterImageFields {
		if source, ok := post.Extra[field.name].(string); ok {
			c.markPrivateSource(source, post)
		}
	}
}

func (c *catalog) markPrivateSource(rawSrc string, post *models.Post) {
	rawSrc = strings.TrimSpace(rawSrc)
	if rawSrc == "" || ignoredSource(rawSrc) {
		return
	}
	if localPath := c.resolveLocalPath(rawSrc, post); localPath != "" {
		c.privateLocals[localPath] = struct{}{}
		c.privateLocals[canonicalLocalPath(localPath)] = struct{}{}
		return
	}
	for _, candidate := range c.localCandidates(rawSrc, post) {
		if c.pathContainsSymlink(candidate) {
			c.privateLocals[canonicalLocalPath(candidate)] = struct{}{}
		}
	}
}

func (c *catalog) removePrivateOnlyImages() {
	for localPath, imageRecord := range c.byLocal {
		if _, ok := c.privateLocals[localPath]; !ok {
			if _, ok := c.privateLocals[canonicalLocalPath(localPath)]; !ok {
				continue
			}
		}
		if len(imageRecord.uses) > 0 {
			continue
		}
		delete(c.byLocal, localPath)
		delete(c.byKey, "local:"+imageRecord.local)
	}
}

func (c *catalog) addFrontmatterImages(post *models.Post) {
	if post == nil || post.Extra == nil {
		return
	}

	coverFound := false
	for _, field := range frontmatterImageFields {
		src, ok := post.Extra[field.name].(string)
		if !ok || strings.TrimSpace(src) == "" {
			continue
		}
		isCover := field.cover && !coverFound
		if isCover {
			coverFound = true
		}
		alt := frontmatterAlt(post.Extra, field.altKeys...)
		c.addReference(imageReference{src: src, alt: alt}, post, isCover)
	}
}

type frontmatterImageField struct {
	name    string
	cover   bool
	altKeys []string
}

var frontmatterImageFields = []frontmatterImageField{
	{name: "cover", cover: true, altKeys: []string{"cover_alt", "image_alt", "alt"}},
	{name: "cover_image", cover: true, altKeys: []string{"cover_image_alt", "cover_alt", "image_alt", "alt"}},
	{name: "image", cover: true, altKeys: []string{"image_alt", "alt"}},
	{name: "video", cover: true, altKeys: []string{"video_alt", "image_alt", "alt"}},
	{name: "og_image", altKeys: []string{"og_image_alt", "image_alt", "alt"}},
	{name: "social_image", altKeys: []string{"social_image_alt", "image_alt", "alt"}},
	{name: "thumbnail", altKeys: []string{"thumbnail_alt", "image_alt", "alt"}},
	{name: "featured_image", altKeys: []string{"featured_image_alt", "image_alt", "alt"}},
	{name: "hero_image", altKeys: []string{"hero_image_alt", "image_alt", "alt"}},
	{name: "avatar", altKeys: []string{"avatar_alt", "alt"}},
	{name: "author_image", altKeys: []string{"author_image_alt", "alt"}},
}

func frontmatterAlt(extra map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := extra[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (c *catalog) ensure(key, src string) *catalogImage {
	if existing, ok := c.byKey[key]; ok {
		return existing
	}
	imageRecord := &catalogImage{
		image: Image{Src: src, Uses: []Use{}},
		uses:  make(map[string]Use),
	}
	c.byKey[key] = imageRecord
	return imageRecord
}

func (c *catalog) localSource(localPath, fallback string) string {
	relative, err := filepath.Rel(c.assetsDir, localPath)
	if err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
		return "/" + filepath.ToSlash(relative)
	}
	return localURL(fallback)
}

func imagePostsInStableOrder(posts []*models.Post) []*models.Post {
	ordered := append([]*models.Post(nil), posts...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return imagePostSortKey(ordered[i]) < imagePostSortKey(ordered[j])
	})
	return ordered
}

func imagePostSortKey(post *models.Post) string {
	if post == nil {
		return ""
	}
	extra := ""
	if serialized, err := json.Marshal(post.Extra); err == nil {
		extra = string(serialized)
	}
	return strings.Join([]string{
		post.Path,
		post.Slug,
		post.Href,
		post.PlainTitle(),
		post.InputHash,
		post.Content,
		post.ArticleHTML,
		post.RawFrontmatter,
		extra,
	}, "\x00")
}

func (c *catalog) populateLocalMetadata(imageRecord *catalogImage, path string) {
	if imageRecord == nil {
		return
	}
	if imageRecord.image.MIMEType == "" {
		imageRecord.image.MIMEType = mimeType(path)
	}
	if imageRecord.image.Width == 0 || imageRecord.image.Height == 0 {
		imageRecord.image.Width, imageRecord.image.Height = dimensions(path)
	}
	if imageRecord.image.Alt == "" {
		c.setAlt(imageRecord, fallbackAlt(imageRecord.image.Src), 1)
	}
}

func (c *catalog) setAlt(imageRecord *catalogImage, alt string, rank int) {
	alt = strings.TrimSpace(alt)
	if imageRecord == nil || alt == "" || rank < imageRecord.altRank {
		return
	}
	if rank == imageRecord.altRank && imageRecord.image.Alt != "" {
		return
	}
	imageRecord.image.Alt = alt
	imageRecord.altRank = rank
}

func (c *catalog) addUse(imageRecord *catalogImage, post *models.Post, cover, embed bool, caption string) {
	if imageRecord == nil || post == nil {
		return
	}
	if post.Date != nil && !post.Date.IsZero() {
		postDate := post.Date.UTC()
		if imageRecord.image.AddedAt == nil || postDate.Before(*imageRecord.image.AddedAt) {
			imageRecord.image.AddedAt = &postDate
		}
		if imageRecord.image.LastUsedAt == nil || imageRecord.image.LastUsedAt.Before(postDate) {
			imageRecord.image.LastUsedAt = &postDate
		}
	}
	href := strings.TrimSpace(post.Href)
	if href == "" && post.Slug != "" {
		href = "/" + strings.Trim(post.Slug, "/") + "/"
	}
	postPath := strings.TrimSpace(post.Path)
	if postPath == "" {
		postPath = post.Slug
	}
	title := post.PlainTitle()
	if title == "" {
		title = post.Slug
	}
	use := Use{Post: postPath, Href: href, Title: title, Caption: strings.TrimSpace(caption), Cover: cover, Embed: embed}
	key := postPath + "\x00" + href
	if existing, ok := imageRecord.uses[key]; ok {
		existing.Cover = existing.Cover || cover
		existing.Embed = existing.Embed || embed
		if existing.Title == "" {
			existing.Title = title
		}
		if existing.Caption == "" {
			existing.Caption = use.Caption
		}
		imageRecord.uses[key] = existing
	} else {
		imageRecord.uses[key] = use
	}
	if cover {
		imageRecord.image.Cover = true
	}
	if embed {
		imageRecord.image.Embed = true
	}
}

func (c *catalog) index(generator Generator) Index {
	images := make([]Image, 0, len(c.byKey))
	for _, imageRecord := range c.byKey {
		image := imageRecord.image
		image.Uses = make([]Use, 0, len(imageRecord.uses))
		for _, use := range imageRecord.uses {
			image.Uses = append(image.Uses, use)
		}
		sort.SliceStable(image.Uses, func(i, j int) bool {
			if image.Uses[i].Post != image.Uses[j].Post {
				return image.Uses[i].Post < image.Uses[j].Post
			}
			return image.Uses[i].Href < image.Uses[j].Href
		})
		images = append(images, image)
	}
	return Index{Schema: Schema, SchemaVersion: CurrentVersion, Generator: generator, Images: images}
}

func (c *catalog) resolveLocalPath(rawSrc string, post *models.Post) string {
	if c.rejectedLocalSource(rawSrc, post) {
		return ""
	}
	for _, candidate := range c.localCandidates(rawSrc, post) {
		if c.pathContainsSymlink(candidate) {
			c.privateLocals[canonicalLocalPath(candidate)] = struct{}{}
			continue
		}
		info, err := os.Lstat(candidate)
		if err == nil && info.Mode().IsRegular() && isMediaPath(candidate) {
			return candidate
		}
	}
	return ""
}

func (c *catalog) rejectedLocalSource(rawSrc string, post *models.Post) bool {
	for _, candidate := range c.localCandidates(rawSrc, post) {
		if c.pathContainsSymlink(candidate) {
			return true
		}
	}
	return false
}

func (c *catalog) pathContainsSymlink(path string) bool {
	for _, root := range []string{c.assetsDir, c.contentDir} {
		rootAbsolute, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		pathAbsolute, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		relative, err := filepath.Rel(filepath.Clean(rootAbsolute), filepath.Clean(pathAbsolute))
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			continue
		}
		current := filepath.Clean(rootAbsolute)
		for _, component := range strings.Split(relative, string(filepath.Separator)) {
			current = filepath.Join(current, component)
			info, err := os.Lstat(current)
			if os.IsNotExist(err) {
				break
			}
			if err == nil && info.Mode()&os.ModeSymlink != 0 {
				return true
			}
		}
	}
	return false
}

func (c *catalog) localCandidates(rawSrc string, post *models.Post) []string {
	u, err := url.Parse(rawSrc)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return nil
	}
	pathValue, err := url.PathUnescape(u.Path)
	if err != nil {
		pathValue = u.Path
	}
	pathValue = filepath.FromSlash(pathValue)
	if pathValue == "" {
		return nil
	}

	trimmed := strings.TrimPrefix(pathValue, string(filepath.Separator))
	if strings.HasPrefix(filepath.ToSlash(trimmed), "static/") {
		trimmed = filepath.FromSlash(strings.TrimPrefix(filepath.ToSlash(trimmed), "static/"))
	}
	candidates := make([]string, 0, 4)
	if candidate := pathWithinRoot(c.assetsDir, filepath.Join(c.assetsDir, trimmed)); candidate != "" {
		candidates = append(candidates, candidate)
	}
	if post != nil && post.Path != "" {
		postPath := post.Path
		if !filepath.IsAbs(postPath) {
			postPath = filepath.Join(c.contentDir, postPath)
		}
		if candidate := pathWithinRoot(c.contentDir, filepath.Join(filepath.Dir(postPath), pathValue)); candidate != "" {
			candidates = append(candidates, candidate)
		}
	}
	if candidate := pathWithinRoot(c.contentDir, filepath.Join(c.contentDir, trimmed)); candidate != "" {
		candidates = append(candidates, candidate)
	}
	return candidates
}

func pathWithinRoot(root, path string) string {
	rootAbsolute, err := filepath.Abs(root)
	if err != nil {
		return ""
	}
	pathAbsolute, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	relative, err := filepath.Rel(filepath.Clean(rootAbsolute), filepath.Clean(pathAbsolute))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return ""
	}
	return filepath.Clean(pathAbsolute)
}

func canonicalLocalPath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = filepath.Clean(path)
	}
	if evaluated, err := filepath.EvalSymlinks(absolute); err == nil {
		absolute = evaluated
	}
	return filepath.Clean(absolute)
}

func localURL(rawSrc string) string {
	u, err := url.Parse(rawSrc)
	if err != nil || u.Path == "" {
		return rawSrc
	}
	if strings.HasPrefix(u.Path, "/") {
		return u.Path
	}
	return "/" + strings.TrimPrefix(u.Path, "./")
}

func isPublicPost(post *models.Post) bool {
	return post != nil && post.Published && !post.Draft && !post.Private && !post.Skip
}

func ignoredSource(rawSrc string) bool {
	lower := strings.ToLower(strings.TrimSpace(rawSrc))
	if strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "blob:") {
		return true
	}
	u, err := url.Parse(lower)
	if err != nil {
		return false
	}
	switch u.Scheme {
	case "javascript", "vbscript":
		return true
	default:
		return false
	}
}

const (
	mediaExtensionBMP  = ".bmp"
	mediaExtensionICO  = ".ico"
	mediaExtensionSVG  = ".svg"
	mediaExtensionTIF  = ".tif"
	mediaExtensionTIFF = ".tiff"
	mediaExtensionWEBP = ".webp"
)

func isImagePath(path string) bool {
	ext := mediaExtension(path)
	switch ext {
	case ".avif", mediaExtensionBMP, ".gif", ".heic", ".heif", mediaExtensionICO, ".jpeg", ".jpg", ".png", mediaExtensionSVG, mediaExtensionTIF, mediaExtensionTIFF, mediaExtensionWEBP:
		return true
	default:
		return false
	}
}

func isMediaPath(path string) bool {
	return isImagePath(path) || templates.IsVideoURL(path)
}

func mimeType(path string) string {
	if video := templates.VideoMIMEType(path); video != "" {
		return video
	}
	switch mediaExtension(path) {
	case ".avif":
		return "image/avif"
	case mediaExtensionBMP:
		return "image/bmp"
	case ".gif":
		return "image/gif"
	case ".heic":
		return "image/heic"
	case ".heif":
		return "image/heif"
	case mediaExtensionICO:
		return "image/x-icon"
	case ".jpeg", ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case mediaExtensionSVG:
		return "image/svg+xml"
	case mediaExtensionTIF, mediaExtensionTIFF:
		return "image/tiff"
	case mediaExtensionWEBP:
		return "image/webp"
	default:
		return ""
	}
}

func mediaExtension(path string) string {
	u, err := url.Parse(path)
	if err == nil && u.Path != "" {
		return strings.ToLower(filepath.Ext(u.Path))
	}
	return strings.ToLower(filepath.Ext(path))
}

func dimensions(path string) (width, height int) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var config image.Config
	switch mediaExtension(path) {
	case mediaExtensionBMP:
		config, err = bmp.DecodeConfig(file)
	case mediaExtensionTIF, mediaExtensionTIFF:
		config, err = tiff.DecodeConfig(file)
	case mediaExtensionWEBP:
		// x/image/webp supports VP8, VP8L, and VP8X containers. Call it
		// directly because image.DecodeConfig's format sniff is limited to
		// the common VP8 signature.
		config, err = webp.DecodeConfig(file)
	case mediaExtensionICO:
		return icoDimensions(file)
	case mediaExtensionSVG:
		return svgDimensions(file)
	default:
		config, _, err = image.DecodeConfig(file)
	}
	if err != nil {
		return 0, 0
	}
	return config.Width, config.Height
}

const maxSVGMetadataBytes = 1 << 20

// svgDimensions reads only the SVG root attributes needed for a useful
// intrinsic size. Percentages are not absolute dimensions; a viewBox is used
// when it supplies a positive fallback.
func svgDimensions(file *os.File) (width, height int) {
	decoder := xml.NewDecoder(io.LimitReader(file, maxSVGMetadataBytes))
	for {
		token, err := decoder.Token()
		if err != nil {
			return 0, 0
		}
		start, ok := token.(xml.StartElement)
		if !ok || !strings.EqualFold(start.Name.Local, "svg") {
			continue
		}

		var widthValue, heightValue, viewBox string
		for _, attribute := range start.Attr {
			switch strings.ToLower(attribute.Name.Local) {
			case "width":
				widthValue = attribute.Value
			case "height":
				heightValue = attribute.Value
			case "viewbox":
				viewBox = attribute.Value
			}
		}
		return svgIntrinsicDimensions(widthValue, heightValue, viewBox)
	}
}

func svgIntrinsicDimensions(widthValue, heightValue, viewBox string) (width, height int) {
	widthValuePixels, widthOK := svgLength(widthValue)
	heightValuePixels, heightOK := svgLength(heightValue)
	viewBoxWidth, viewBoxHeight, viewBoxOK := svgViewBoxDimensions(viewBox)

	if widthOK && heightOK {
		width, widthOK = positiveDimension(widthValuePixels)
		height, heightOK = positiveDimension(heightValuePixels)
		if widthOK && heightOK {
			return width, height
		}
	}
	if viewBoxOK {
		if widthOK {
			if derived, ok := positiveDimension(widthValuePixels * viewBoxHeight / viewBoxWidth); ok {
				width, height = positiveDimensionPair(widthValuePixels, float64(derived))
				if width > 0 && height > 0 {
					return width, height
				}
			}
		}
		if heightOK {
			if derived, ok := positiveDimension(heightValuePixels * viewBoxWidth / viewBoxHeight); ok {
				width, height = positiveDimensionPair(float64(derived), heightValuePixels)
				if width > 0 && height > 0 {
					return width, height
				}
			}
		}
		return positiveDimensionPair(viewBoxWidth, viewBoxHeight)
	}
	return 0, 0
}

func svgLength(raw string) (float64, bool) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" || strings.HasSuffix(raw, "%") {
		return 0, false
	}
	unitFactor := 1.0
	for _, unit := range []struct {
		suffix string
		factor float64
	}{
		{suffix: "px", factor: 1},
		{suffix: "in", factor: 96},
		{suffix: "cm", factor: 96 / 2.54},
		{suffix: "mm", factor: 96 / 25.4},
		{suffix: "pt", factor: 96 / 72},
		{suffix: "pc", factor: 16},
	} {
		if strings.HasSuffix(raw, unit.suffix) {
			raw = strings.TrimSpace(strings.TrimSuffix(raw, unit.suffix))
			unitFactor = unit.factor
			break
		}
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0, false
	}
	return value * unitFactor, true
}

func svgViewBoxDimensions(raw string) (width, height float64, ok bool) {
	values := strings.Fields(strings.NewReplacer(",", " ").Replace(strings.TrimSpace(raw)))
	if len(values) != 4 {
		return 0, 0, false
	}
	width, widthErr := strconv.ParseFloat(values[2], 64)
	height, heightErr := strconv.ParseFloat(values[3], 64)
	if widthErr != nil || heightErr != nil || math.IsNaN(width) || math.IsNaN(height) || math.IsInf(width, 0) || math.IsInf(height, 0) || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func positiveDimension(value float64) (int, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0, false
	}
	maxInt := int(^uint(0) >> 1)
	value = math.Round(value)
	// float64(maxInt) rounds up to the next power of two on common 64-bit
	// platforms. Reject that boundary before converting so a huge SVG value
	// cannot wrap into a negative int and fail index normalization.
	if value < 1 || value >= float64(maxInt) {
		return 0, false
	}
	return int(value), true
}

func positiveDimensionPair(width, height float64) (parsedWidth, parsedHeight int) {
	parsedWidth, widthOK := positiveDimension(width)
	parsedHeight, heightOK := positiveDimension(height)
	if !widthOK || !heightOK {
		return 0, 0
	}
	return parsedWidth, parsedHeight
}

func icoDimensions(file *os.File) (width, height int) {
	var header [6]byte
	if _, err := io.ReadFull(file, header[:]); err != nil || binary.LittleEndian.Uint16(header[0:2]) != 0 || binary.LittleEndian.Uint16(header[2:4]) != 1 {
		return 0, 0
	}
	count := binary.LittleEndian.Uint16(header[4:6])
	if count == 0 || count > 4096 {
		return 0, 0
	}

	bestArea := uint64(0)
	for index := 0; index < int(count); index++ {
		var entry [16]byte
		if _, err := io.ReadFull(file, entry[:]); err != nil {
			return 0, 0
		}
		entryWidth, entryHeight := int(entry[0]), int(entry[1])
		if entryWidth == 0 {
			entryWidth = 256
		}
		if entryHeight == 0 {
			entryHeight = 256
		}
		areaWidth, areaHeight := uint64(entry[0]), uint64(entry[1])
		if areaWidth == 0 {
			areaWidth = 256
		}
		if areaHeight == 0 {
			areaHeight = 256
		}
		area := areaWidth * areaHeight
		if area > bestArea {
			bestArea = area
			width, height = entryWidth, entryHeight
		}
	}
	return width, height
}

var markdownParser = goldmark.New(
	goldmark.WithExtensions(extension.GFM, figure.Figure),
	goldmark.WithParserOptions(parser.WithAttribute()),
)

func extractMarkdownImages(content string) []imageReference {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	source := []byte(content)
	document := markdownParser.Parser().Parse(text.NewReader(source))
	result := make([]imageReference, 0, 4)
	if err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		imageNode, ok := node.(*ast.Image)
		if !ok || len(imageNode.Destination) == 0 {
			return ast.WalkContinue, nil
		}
		result = append(result, imageReference{
			src:     string(imageNode.Destination),
			alt:     inlineText(imageNode, source),
			caption: markdownFigureCaption(imageNode, source),
		})
		return ast.WalkContinue, nil
	}); err != nil {
		return nil
	}
	return result
}

func inlineText(node ast.Node, source []byte) string {
	var result strings.Builder
	if err := ast.Walk(node, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch typed := child.(type) {
		case *ast.Text:
			result.Write(typed.Segment.Value(source))
		case *ast.String:
			result.Write(typed.Value)
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return ""
	}
	return strings.Join(strings.Fields(result.String()), " ")
}

func markdownFigureCaption(node ast.Node, source []byte) string {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		figureNode, ok := parent.(*figureast.Figure)
		if !ok {
			continue
		}
		for child := figureNode.FirstChild(); child != nil; child = child.NextSibling() {
			captionNode, ok := child.(*figureast.FigureCaption)
			if ok {
				return inlineText(captionNode, source)
			}
		}
		return ""
	}
	return ""
}

func extractHTMLMedia(source string) []imageReference {
	if strings.TrimSpace(source) == "" {
		return nil
	}
	document, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return nil
	}
	result := make([]imageReference, 0, 4)
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch strings.ToLower(node.Data) {
			case "img":
				src := htmlAttribute(node, "src")
				if strings.TrimSpace(src) != "" {
					video := isVideoSource(src, "")
					result = append(result, imageReference{
						src:     src,
						alt:     htmlMediaAlt(node),
						caption: htmlMediaCaption(node),
						video:   video,
						embed:   htmlNodeIsEmbed(node),
					})
				}
			case "video":
				src := htmlAttribute(node, "src")
				if strings.TrimSpace(src) != "" {
					result = append(result, imageReference{
						src:      src,
						alt:      htmlMediaAlt(node),
						caption:  htmlMediaCaption(node),
						poster:   htmlAttribute(node, "poster"),
						mimeType: htmlAttribute(node, "type"),
						video:    true,
						embed:    htmlNodeIsEmbed(node),
					})
				}
			case "source":
				videoNode := nearestVideoAncestor(node)
				if videoNode != nil {
					src := htmlAttribute(node, "src")
					if strings.TrimSpace(src) != "" {
						mime := htmlAttribute(node, "type")
						if mime == "" {
							mime = htmlAttribute(videoNode, "type")
						}
						result = append(result, imageReference{
							src:      src,
							alt:      htmlMediaAlt(videoNode),
							caption:  htmlMediaCaption(node),
							poster:   htmlAttribute(videoNode, "poster"),
							mimeType: mime,
							video:    true,
							embed:    htmlNodeIsEmbed(node),
						})
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	return result
}

func htmlAttribute(node *html.Node, key string) string {
	if node == nil {
		return ""
	}
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, key) {
			return strings.TrimSpace(attribute.Val)
		}
	}
	return ""
}

func htmlMediaAlt(node *html.Node) string {
	for _, key := range []string{"alt", "aria-label", "title"} {
		if value := htmlAttribute(node, key); value != "" {
			return value
		}
	}
	return ""
}

func htmlMediaCaption(node *html.Node) string {
	for current := node; current != nil; current = current.Parent {
		if current.Type != html.ElementNode || !strings.EqualFold(current.Data, "figure") {
			continue
		}
		caption := htmlDescendantElement(current, "figcaption")
		if caption == nil {
			return ""
		}
		return htmlNodeText(caption)
	}
	return ""
}

func htmlDescendantElement(node *html.Node, name string) *html.Node {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && strings.EqualFold(child.Data, name) {
			return child
		}
		if child.Type == html.ElementNode && strings.EqualFold(child.Data, "figure") {
			continue
		}
		if descendant := htmlDescendantElement(child, name); descendant != nil {
			return descendant
		}
	}
	return nil
}

func htmlNodeText(node *html.Node) string {
	var result strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			result.WriteString(current.Data)
			return
		}
		if current.Type == html.ElementNode && (strings.EqualFold(current.Data, "script") || strings.EqualFold(current.Data, "style")) {
			return
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.Join(strings.Fields(result.String()), " ")
}

func nearestVideoAncestor(node *html.Node) *html.Node {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if parent.Type == html.ElementNode && strings.EqualFold(parent.Data, "video") {
			return parent
		}
	}
	return nil
}

func htmlNodeIsEmbed(node *html.Node) bool {
	for current := node; current != nil; current = current.Parent {
		if current.Type != html.ElementNode {
			continue
		}
		for _, attribute := range current.Attr {
			if strings.EqualFold(attribute.Key, "data-markata-embed") {
				value := strings.ToLower(strings.TrimSpace(attribute.Val))
				if value == "true" || value == "external" {
					return true
				}
			}
		}
	}
	return false
}

func fallbackAlt(rawSrc string) string {
	u, err := url.Parse(rawSrc)
	pathValue := rawSrc
	if err == nil && u.Path != "" {
		pathValue = u.Path
	}
	pathValue, err = url.PathUnescape(pathValue)
	if err != nil {
		pathValue = strings.TrimSpace(pathValue)
	}
	base := filepath.Base(filepath.FromSlash(pathValue))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(base)
	return strings.Join(strings.Fields(base), " ")
}
