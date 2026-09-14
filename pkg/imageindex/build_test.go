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
	for _, image := range index.Images {
		if image.Src == src {
			return image, true
		}
	}
	return Image{}, false
}

func stringPointer(value string) *string {
	return &value
}
