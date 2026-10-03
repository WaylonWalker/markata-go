package plugins

import (
	"html"
	"strings"

	htmlparser "golang.org/x/net/html"
)

func markPostFontpack(content, name string) string {
	if content == "" {
		return content
	}
	content = markHTMLFontpack(content, name)
	if !strings.Contains(content, `href="/css/fonts.`) {
		content = strings.Replace(content, "</head>", `  <link rel="stylesheet" href="/css/fonts.css">`+"\n</head>", 1)
	}
	return content
}

func markHTMLFontpack(content, name string) string {
	tokenizer := htmlparser.NewTokenizer(strings.NewReader(content))
	offset := 0
	for {
		token := tokenizer.Next()
		start := offset
		offset += len(tokenizer.Raw())
		switch token {
		case htmlparser.ErrorToken:
			return content
		case htmlparser.StartTagToken, htmlparser.SelfClosingTagToken:
			tag, _ := tokenizer.TagName()
			if !strings.EqualFold(string(tag), "html") {
				continue
			}
			open := content[start:offset]
			marked := markFontpackStartTag(open, name, token == htmlparser.SelfClosingTagToken)
			if marked == open {
				return content
			}
			return content[:start] + marked + content[offset:]
		default:
		}
	}
}

type fontpackAttribute struct {
	start    int
	end      int
	value    string
	hasValue bool
}

func markFontpackStartTag(open, name string, selfClosing bool) string {
	var attributes []fontpackAttribute
	end := len(open) - 1
	for i := len("<html"); i < end; {
		for i < end && fontpackHTMLSpace(open[i]) {
			i++
		}
		if i == end || (open[i] == '/' && i == end-1) {
			break
		}
		key, attribute, next := scanFontpackAttribute(open, i, end)
		i = next
		if strings.EqualFold(key, "data-fontpack") {
			attributes = append(attributes, attribute)
		}
	}
	if len(attributes) == 1 && attributes[0].hasValue && html.UnescapeString(attributes[0].value) == name {
		return open
	}

	replacement := `data-fontpack="` + html.EscapeString(name) + `"`
	if len(attributes) == 0 {
		insert := end
		if selfClosing {
			insert--
		}
		return open[:insert] + " " + replacement + open[insert:]
	}
	var marked strings.Builder
	marked.Grow(len(open) + len(replacement))
	cursor := 0
	for i, attribute := range attributes {
		marked.WriteString(open[cursor:attribute.start])
		if i == 0 {
			marked.WriteString(replacement)
		}
		cursor = attribute.end
	}
	marked.WriteString(open[cursor:])
	return marked.String()
}

func scanFontpackAttribute(open string, start, end int) (key string, attribute fontpackAttribute, next int) {
	if open[start] == '/' {
		return "", fontpackAttribute{}, start + 1
	}
	i := start + 1
	for i < end && !fontpackHTMLSpace(open[i]) && open[i] != '=' && open[i] != '/' {
		i++
	}
	key = open[start:i]
	attribute = fontpackAttribute{start: start, end: i}
	for i < end && fontpackHTMLSpace(open[i]) {
		i++
	}
	if i < end && open[i] == '=' {
		i++
		for i < end && fontpackHTMLSpace(open[i]) {
			i++
		}
		attribute.hasValue = true
		attribute.value, i = scanFontpackValue(open, i, end)
		attribute.end = i
	}
	return key, attribute, i
}

func scanFontpackValue(open string, start, end int) (value string, next int) {
	if start < end && (open[start] == '"' || open[start] == '\'') {
		quote := open[start]
		start++
		i := start
		for i < end && open[i] != quote {
			i++
		}
		value = open[start:i]
		if i < end {
			i++
		}
		return value, i
	}
	i := start
	for i < end && !fontpackHTMLSpace(open[i]) {
		i++
	}
	return open[start:i], i
}

func fontpackHTMLSpace(value byte) bool {
	switch value {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}
