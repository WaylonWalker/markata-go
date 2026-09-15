package imageindex

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestBuild_DiscoversPublicSourcesAndDeduplicatesUses(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "static")
	if err := os.MkdirAll(filepath.Join(assets, "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	writePNG(t, filepath.Join(assets, "images", "photo.png"), 3, 2)
	writePNG(t, filepath.Join(assets, "images", "unused.png"), 4, 4)
	writePNG(t, filepath.Join(assets, "images", "private.png"), 5, 5)

	public := models.NewPost(filepath.Join(root, "posts", "hello.md"))
	public.Path = "posts/hello.md"
	public.Slug = "hello"
	public.Href = "/hello/"
	public.Published = true
	public.Content = "![Authored photo](images/photo.png)\n\n<img src=\"/static/images/photo.png\" alt=\"HTML photo\">\n\n![Inline](data:image/png;base64,abc) ![Blob](blob:abc) ![Script](javascript:alert(1))"
	public.ArticleHTML = `<p><img src="/images/photo.png" alt="Rendered photo"></p>`
	public.Extra["cover"] = "/static/images/photo.png"
	public.Extra["cover_image"] = "https://cdn.example.test/second.webp"
	public.Title = stringPointer("Hello")

	private := models.NewPost("posts/private.md")
	private.Path = "posts/private.md"
	private.Published = true
	private.Private = true
	private.Content = "![Secret](secret.png) ![Private asset](/images/private.png)"

	index, err := Build([]*models.Post{public, private}, BuildOptions{
		ContentDir:          root,
		AssetsDir:           assets,
		GeneratorVersion:    "test",
		IncludeUnreferenced: true,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if index.ImageCount != 3 {
		t.Fatalf("ImageCount = %d, want 3 (%#v)", index.ImageCount, index.Images)
	}

	photo := imageBySrc(t, index, "/images/photo.png")
	if photo.Width != 3 || photo.Height != 2 || photo.MIMEType != "image/png" {
		t.Fatalf("photo metadata = %#v", photo)
	}
	if !photo.Cover || len(photo.Uses) != 1 || !photo.Uses[0].Cover {
		t.Fatalf("photo uses = %#v", photo.Uses)
	}
	if photo.Alt != "Authored photo" {
		t.Fatalf("photo alt = %q, want first authored alt", photo.Alt)
	}
	if len(imageBySrc(t, index, "https://cdn.example.test/second.webp").Uses) != 1 {
		t.Fatal("frontmatter cover_image was not indexed")
	}
	if len(imageBySrc(t, index, "https://cdn.example.test/second.webp").Uses) != 1 || imageBySrc(t, index, "https://cdn.example.test/second.webp").Cover {
		t.Fatal("cover_image should be a non-cover relationship when cover is present")
	}
	if len(imageBySrc(t, index, "/images/unused.png").Uses) != 0 {
		t.Fatal("unreferenced image has unexpected public uses")
	}
	if _, ok := findImage(index, "secret.png"); ok {
		t.Fatal("private image leaked into index")
	}
	if _, ok := findImage(index, "/images/private.png"); ok {
		t.Fatal("private-only static image leaked into index")
	}
	for _, source := range []string{"data:image/png;base64,abc", "blob:abc", "javascript:alert(1)"} {
		if _, ok := findImage(index, source); ok {
			t.Fatalf("unsafe image source %q leaked into index", source)
		}
	}
}

func TestBuild_OnlyReferencedImagesWhenUnreferencedDisabled(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "static")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	writePNG(t, filepath.Join(assets, "only-static.png"), 1, 1)

	post := models.NewPost("posts/one.md")
	post.Path = "posts/one.md"
	post.Slug = "one"
	post.Published = true
	post.Content = "![Referenced](https://example.test/remote.jpg)"

	index, err := Build([]*models.Post{post}, BuildOptions{ContentDir: root, AssetsDir: assets})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if index.ImageCount != 1 || index.Images[0].Src != "https://example.test/remote.jpg" {
		t.Fatalf("Build() = %#v", index.Images)
	}
}

func TestBuild_ScansUnreferencedVideoAssets(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "static", "videos")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	videoPath := filepath.Join(assets, "clip.mp4")
	if err := os.WriteFile(videoPath, []byte("test video"), 0o600); err != nil {
		t.Fatal(err)
	}

	index, err := Build(nil, BuildOptions{
		ContentDir:          root,
		AssetsDir:           filepath.Join(root, "static"),
		IncludeUnreferenced: true,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	video := imageBySrc(t, index, "/videos/clip.mp4")
	if video.MIMEType != "video/mp4" || video.Cover || len(video.Uses) != 0 {
		t.Fatalf("unreferenced video = %#v", video)
	}
}

func TestBuild_ClassifiesCoversEmbedsAndVideos(t *testing.T) {
	root := t.TempDir()
	post := models.NewPost(filepath.Join(root, "posts", "media.md"))
	post.Path = "posts/media.md"
	post.Slug = "media"
	post.Href = "/media/"
	post.Published = true
	post.Title = stringPointer("Media")
	postDate := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.FixedZone("test", 2*60*60))
	post.Date = &postDate
	post.Extra["image"] = "https://dropper.wayl.one/file/clip.mp4?token=keep"
	post.Extra["poster_image"] = "http://dropper.wayl.one/file/clip.webp"
	post.Content = `![Inline video](https://dropper.wayl.one/file/inline.mp4)

<div class="custom-external-card" data-markata-embed="true"><img src="https://example.test/og-image.jpg" alt="External OG image"></div>`
	post.ArticleHTML = `<video poster="https://dropper.wayl.one/file/body.webp"><source src="https://dropper.wayl.one/file/body.webm" type="video/webm"></video>`

	index, err := Build([]*models.Post{post}, BuildOptions{ContentDir: root, GeneratorVersion: "test"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if index.ImageCount != 4 {
		t.Fatalf("ImageCount = %d, want 4 (%#v)", index.ImageCount, index.Images)
	}

	coverVideo := imageBySrc(t, index, "https://dropper.wayl.one/file/clip.mp4?token=keep")
	if coverVideo.MIMEType != "video/mp4" || !coverVideo.Cover || coverVideo.PosterSrc != "https://dropper.wayl.one/file/clip.webp" || coverVideo.LastUsedAt == nil || !coverVideo.LastUsedAt.Equal(postDate.UTC()) {
		t.Fatalf("frontmatter video = %#v", coverVideo)
	}
	if len(coverVideo.Uses) != 1 || !coverVideo.Uses[0].Cover || coverVideo.Uses[0].Embed {
		t.Fatalf("frontmatter video uses = %#v", coverVideo.Uses)
	}

	bodyVideo := imageBySrc(t, index, "https://dropper.wayl.one/file/body.webm")
	if bodyVideo.MIMEType != "video/webm" || bodyVideo.PosterSrc != "https://dropper.wayl.one/file/clip.webp" {
		t.Fatalf("HTML video = %#v", bodyVideo)
	}

	inlineVideo := imageBySrc(t, index, "https://dropper.wayl.one/file/inline.mp4")
	if inlineVideo.MIMEType != "video/mp4" || inlineVideo.PosterSrc != "https://dropper.wayl.one/file/clip.webp" {
		t.Fatalf("Markdown video = %#v", inlineVideo)
	}

	embedImage := imageBySrc(t, index, "https://example.test/og-image.jpg")
	if !embedImage.Embed || embedImage.Cover || len(embedImage.Uses) != 1 || !embedImage.Uses[0].Embed {
		t.Fatalf("embedded OG image = %#v", embedImage)
	}
}

func TestBuild_DoesNotMarkInternalEmbedMediaAsExternal(t *testing.T) {
	post := models.NewPost("posts/internal.md")
	post.Path = "posts/internal.md"
	post.Slug = "internal"
	post.Href = "/internal/"
	post.Published = true
	post.Content = `<div class="embed-card"><img src="https://example.test/internal.jpg" alt="Internal image"></div>`

	index, err := Build([]*models.Post{post}, BuildOptions{GeneratorVersion: "test"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	image := imageBySrc(t, index, "https://example.test/internal.jpg")
	if image.Embed || len(image.Uses) != 1 || image.Uses[0].Embed {
		t.Fatalf("internal embed media = %#v", image)
	}
}

func TestBuild_ClassifiesDirectHTMLVideoWithPoster(t *testing.T) {
	post := models.NewPost("posts/direct-video.md")
	post.Path = "posts/direct-video.md"
	post.Slug = "direct-video"
	post.Href = "/direct-video/"
	post.Published = true
	post.ArticleHTML = `<video src="https://example.test/direct.mp4" type="video/mp4" poster="https://example.test/direct-poster.jpg"></video>`

	index, err := Build([]*models.Post{post}, BuildOptions{GeneratorVersion: "test"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	video := imageBySrc(t, index, "https://example.test/direct.mp4")
	if video.MIMEType != "video/mp4" || video.PosterSrc != "https://example.test/direct-poster.jpg" {
		t.Fatalf("direct HTML video = %#v", video)
	}
}

func TestBuild_CollectsFigureCaptionsForImagesAndVideos(t *testing.T) {
	post := models.NewPost("posts/captions.md")
	post.Path = "posts/captions.md"
	post.Slug = "captions"
	post.Href = "/captions/"
	post.Published = true
	post.Title = stringPointer("Caption post")
	post.Content = `![Markdown image](https://example.test/markdown.jpg)
![Second image](https://example.test/second.jpg)
Markdown figure caption`
	post.ArticleHTML = `<figure><a href="https://example.test/html.jpg"><img src="https://example.test/html.jpg" alt="HTML image"></a><figcaption><strong>HTML image</strong> caption</figcaption></figure>
<figure><video src="https://example.test/direct-clip.mp4"></video><figcaption>Direct video caption</figcaption></figure>
<figure><video><source src="https://example.test/clip.mp4" type="video/mp4"></video><figcaption>Video <em>figure</em> caption</figcaption></figure>`

	index, err := Build([]*models.Post{post}, BuildOptions{GeneratorVersion: "test"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	for _, test := range []struct {
		src     string
		caption string
	}{
		{src: "https://example.test/markdown.jpg", caption: "Markdown figure caption"},
		{src: "https://example.test/second.jpg", caption: "Markdown figure caption"},
		{src: "https://example.test/html.jpg", caption: "HTML image caption"},
		{src: "https://example.test/direct-clip.mp4", caption: "Direct video caption"},
		{src: "https://example.test/clip.mp4", caption: "Video figure caption"},
	} {
		image := imageBySrc(t, index, test.src)
		if len(image.Uses) != 1 || image.Uses[0].Caption != test.caption {
			t.Fatalf("caption for %q = %#v, want %q", test.src, image.Uses, test.caption)
		}
	}
}

func TestBuild_TracksLatestPublicUse(t *testing.T) {
	olderDate := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	newerDate := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	older := models.NewPost("posts/older.md")
	older.Path = "posts/older.md"
	older.Slug = "older"
	older.Href = "/older/"
	older.Published = true
	older.Date = &olderDate
	older.Content = "![Shared](https://example.test/shared.jpg)"
	newer := models.NewPost("posts/newer.md")
	newer.Path = "posts/newer.md"
	newer.Slug = "newer"
	newer.Href = "/newer/"
	newer.Published = true
	newer.Date = &newerDate
	newer.Content = "![Shared](https://example.test/shared.jpg)"

	index, err := Build([]*models.Post{older, newer}, BuildOptions{GeneratorVersion: "test"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	shared := imageBySrc(t, index, "https://example.test/shared.jpg")
	if shared.LastUsedAt == nil || !shared.LastUsedAt.Equal(newerDate) {
		t.Fatalf("latest use = %#v, want %s", shared.LastUsedAt, newerDate.Format(time.RFC3339))
	}
}

func TestBuild_IsDeterministicAcrossPostOrderAndLocalAliases(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "static", "images")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	writePNG(t, filepath.Join(assets, "photo.png"), 2, 2)

	first := models.NewPost("posts/a.md")
	first.Path = "posts/a.md"
	first.Slug = "a"
	first.Href = "/a/"
	first.Published = true
	first.Content = "![First alt](/static/images/photo.png)"
	first.Title = stringPointer("A")

	second := models.NewPost("posts/b.md")
	second.Path = "posts/b.md"
	second.Slug = "b"
	second.Href = "/b/"
	second.Published = true
	second.Content = "![Second alt](/images/photo.png)"
	second.Title = stringPointer("B")

	build := func(posts []*models.Post) []byte {
		t.Helper()
		index, err := Build(posts, BuildOptions{ContentDir: root, AssetsDir: filepath.Join(root, "static")})
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		data, err := json.Marshal(index)
		if err != nil {
			t.Fatalf("marshal index: %v", err)
		}
		return data
	}

	forward := build([]*models.Post{first, second})
	reverse := build([]*models.Post{second, first})
	if !bytes.Equal(forward, reverse) {
		t.Fatalf("post ordering changed output:\nforward=%s\nreverse=%s", forward, reverse)
	}
	index, err := Build([]*models.Post{second, first}, BuildOptions{ContentDir: root, AssetsDir: filepath.Join(root, "static")})
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Images) != 1 || index.Images[0].Src != "/images/photo.png" || index.Images[0].Alt != "First alt" {
		t.Fatalf("canonical image = %#v", index.Images)
	}
}

func TestBuild_PrivateSymlinkDoesNotExposeTargetAsUnreferenced(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "static", "images")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(assets, "private-target.png")
	alias := filepath.Join(assets, "private-alias.png")
	writePNG(t, target, 2, 2)
	if err := os.Symlink("private-target.png", alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	private := models.NewPost(filepath.Join(root, "posts", "private.md"))
	private.Path = "posts/private.md"
	private.Published = true
	private.Private = true
	private.Content = "![Secret](images/private-alias.png)"

	index, err := Build([]*models.Post{private}, BuildOptions{
		ContentDir:          root,
		AssetsDir:           filepath.Join(root, "static"),
		IncludeUnreferenced: true,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if index.ImageCount != 0 {
		t.Fatalf("private symlink images leaked: %#v", index.Images)
	}
}

func writePNG(t *testing.T, path string, width, height int) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			canvas.Set(x, y, color.RGBA{R: 20, G: 90, B: 70, A: 255})
		}
	}
	if err := png.Encode(file, canvas); err != nil {
		t.Fatal(err)
	}
}

func imageBySrc(t *testing.T, index Index, src string) Image {
	t.Helper()
	image, ok := findImage(index, src)
	if !ok {
		t.Fatalf("image %q not found in %#v", src, index.Images)
	}
	return image
}

func findImage(index Index, src string) (Image, bool) {
	for i := range index.Images {
		image := &index.Images[i]
		if image.Src == src {
			return *image, true
		}
	}
	return Image{}, false
}

func stringPointer(value string) *string {
	return &value
}
