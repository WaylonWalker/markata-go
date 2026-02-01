package csspurge

import (
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// UsedSelectors tracks CSS selectors found in HTML content.
type UsedSelectors struct {
	// Classes maps class names to true (for O(1) lookup)
	Classes map[string]bool
	// IDs maps ID names to true
	IDs map[string]bool
	// Elements maps element/tag names to true
	Elements map[string]bool
	// Attributes maps attribute names to true (for [attr] selectors)
	Attributes map[string]bool
}

// NewUsedSelectors creates an empty UsedSelectors struct.
func NewUsedSelectors() *UsedSelectors {
	return &UsedSelectors{
		Classes:    make(map[string]bool),
		IDs:        make(map[string]bool),
		Elements:   make(map[string]bool),
		Attributes: make(map[string]bool),
	}
}

// Merge combines another UsedSelectors into this one.
func (u *UsedSelectors) Merge(other *UsedSelectors) {
	if other == nil {
		return
	}
	for k := range other.Classes {
		u.Classes[k] = true
	}
	for k := range other.IDs {
		u.IDs[k] = true
	}
	for k := range other.Elements {
		u.Elements[k] = true
	}
	for k := range other.Attributes {
		u.Attributes[k] = true
	}
}

// ScanHTML parses an HTML file and extracts used selectors.
// Results are merged into the provided UsedSelectors struct.
func ScanHTML(htmlPath string, used *UsedSelectors) error {
	f, err := os.Open(htmlPath)
	if err != nil {
		return err
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return err
	}

	// Walk all elements in the document
	doc.Find("*").Each(func(_ int, s *goquery.Selection) {
		// Extract element name
		if node := s.Get(0); node != nil {
			used.Elements[strings.ToLower(node.Data)] = true

			// Extract all attributes
			for _, attr := range node.Attr {
				attrName := strings.ToLower(attr.Key)
				used.Attributes[attrName] = true

				// Extract classes from class attribute
				if attrName == "class" {
					classes := strings.Fields(attr.Val)
					for _, class := range classes {
						used.Classes[class] = true
					}
				}

				// Extract ID
				if attrName == "id" && attr.Val != "" {
					used.IDs[attr.Val] = true
				}
			}
		}
	})

	return nil
}

// ScanHTMLContent parses HTML content from a string and extracts used selectors.
// This is useful for processing HTML that's already in memory.
func ScanHTMLContent(html string, used *UsedSelectors) error {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return err
	}

	doc.Find("*").Each(func(_ int, s *goquery.Selection) {
		if node := s.Get(0); node != nil {
			used.Elements[strings.ToLower(node.Data)] = true

			for _, attr := range node.Attr {
				attrName := strings.ToLower(attr.Key)
				used.Attributes[attrName] = true

				if attrName == "class" {
					classes := strings.Fields(attr.Val)
					for _, class := range classes {
						used.Classes[class] = true
					}
				}

				if attrName == "id" && attr.Val != "" {
					used.IDs[attr.Val] = true
				}
			}
		}
	})

	return nil
}
