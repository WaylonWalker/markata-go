package templates

import (
	"strings"

	"github.com/flosch/pongo2/v6"
	"golang.org/x/net/html"
)

// filterPinPreview derives card media and authored notes from existing rendered
// post data. It never fetches metadata or changes the post itself.
func filterPinPreview(in, _ *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	data := valueToStringMap(in)
	image := ""
	for _, key := range []string{"image", "cover", "cover_image", "og_image"} {
		if image = stringFromMap(data, key); image != "" {
			break
		}
	}
	commentary := ""
	if article := stringFromMap(data, "article_html"); article != "" {
		root, err := html.Parse(strings.NewReader(article))
		if err == nil {
			if image == "" {
				media := pinMedia{}
				media.walk(root, false)
				image = media.image()
			}
			paragraphs := []string{}
			collectPinCommentary(root, &paragraphs)
			commentary = strings.Join(paragraphs, "\n\n")
		}
	}
	if commentary == "" {
		commentary = pinNoteText(stringFromMap(data, "description"))
	}
	return pongo2.AsValue(map[string]interface{}{"image": image, "commentary": commentary}), nil
}

type pinMedia struct {
	embed   string
	youtube string
	body    string
}

func (m *pinMedia) image() string {
	for _, image := range []string{m.embed, m.youtube, m.body} {
		if image != "" {
			return image
		}
	}
	return ""
}

func (m *pinMedia) walk(node *html.Node, inEmbed bool) {
	if hasPinClass(node, "embed-card") {
		inEmbed = true
	}
	if node.Type == html.ElementNode && node.Data == "img" && !isPinAvatar(node) {
		if src := pinAttribute(node, "src"); src != "" {
			if inEmbed && m.embed == "" {
				m.embed = src
			} else if m.body == "" {
				m.body = src
			}
		}
	}
	if node.Type == html.ElementNode && node.Data == "lite-youtube" && m.youtube == "" {
		if id := pinAttribute(node, "videoid"); validPinYouTubeID(id) {
			m.youtube = "https://i.ytimg.com/vi/" + id + "/hqdefault.jpg"
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		m.walk(child, inEmbed)
	}
}

func validPinYouTubeID(id string) bool {
	if len(id) != 11 {
		return false
	}
	for _, c := range id {
		if c != '-' && c != '_' && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func collectPinCommentary(node *html.Node, paragraphs *[]string) {
	if skipPinCommentary(node) {
		return
	}
	if node.Type == html.ElementNode && (node.Data == "p" || node.Data == "li") {
		var text strings.Builder
		writePinText(node, &text)
		if note := pinNoteText(text.String()); note != "" {
			*paragraphs = append(*paragraphs, note)
		}
		return
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		collectPinCommentary(child, paragraphs)
	}
}

func writePinText(node *html.Node, text *strings.Builder) {
	if skipPinCommentary(node) {
		return
	}
	if node.Type == html.TextNode {
		text.WriteString(node.Data)
	}
	if node.Type == html.ElementNode && (node.Data == "br" || node.Data == "p" || node.Data == "li") {
		text.WriteByte(' ')
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		writePinText(child, text)
	}
}

func skipPinCommentary(node *html.Node) bool {
	if hasPinClass(node, "embed-card") || isPinAvatar(node) {
		return true
	}
	if node.Type != html.ElementNode {
		return false
	}
	switch node.Data {
	case "script", "style", "pre", "figure", "img", "video", "iframe", "lite-youtube":
		return true
	default:
		return false
	}
}

func pinNoteText(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	url := strings.TrimPrefix(text, "!")
	if !strings.ContainsAny(url, " \t\n") && (strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		return ""
	}
	return text
}

func pinAttribute(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return strings.TrimSpace(attr.Val)
		}
	}
	return ""
}

func hasPinClass(node *html.Node, class string) bool {
	for _, value := range strings.Fields(pinAttribute(node, "class")) {
		if value == class {
			return true
		}
	}
	return false
}

func isPinAvatar(node *html.Node) bool {
	return hasPinClass(node, "link-avatar") || hasPinClass(node, "avatar")
}
