package csspurge

import (
	"strings"
	"testing"
)

func TestParseCSS(t *testing.T) {
	tests := []struct {
		name      string
		css       string
		wantRules int
		checkFunc func(t *testing.T, rules []CSSRule)
	}{
		{
			name:      "single rule",
			css:       `.foo { color: red; }`,
			wantRules: 1,
			checkFunc: func(t *testing.T, rules []CSSRule) {
				if rules[0].Selector != ".foo" {
					t.Errorf("expected selector '.foo', got %q", rules[0].Selector)
				}
			},
		},
		{
			name:      "multiple rules",
			css:       `.foo { color: red; } #bar { color: blue; } div { margin: 0; }`,
			wantRules: 3,
		},
		{
			name:      "rule with multiple selectors",
			css:       `h1, h2, h3 { font-weight: bold; }`,
			wantRules: 1,
			checkFunc: func(t *testing.T, rules []CSSRule) {
				if !strings.Contains(rules[0].Selector, "h1") {
					t.Error("selector should contain h1")
				}
			},
		},
		{
			name:      "media query",
			css:       `@media (max-width: 600px) { .mobile { display: block; } }`,
			wantRules: 1,
			checkFunc: func(t *testing.T, rules []CSSRule) {
				if !rules[0].IsAtRule {
					t.Error("expected @-rule")
				}
				if rules[0].AtRuleType != "media" {
					t.Errorf("expected media, got %q", rules[0].AtRuleType)
				}
				if len(rules[0].NestedRules) != 1 {
					t.Errorf("expected 1 nested rule, got %d", len(rules[0].NestedRules))
				}
			},
		},
		{
			name:      "keyframes",
			css:       `@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`,
			wantRules: 1,
			checkFunc: func(t *testing.T, rules []CSSRule) {
				if rules[0].AtRuleType != "keyframes" {
					t.Errorf("expected keyframes, got %q", rules[0].AtRuleType)
				}
			},
		},
		{
			name:      "font-face",
			css:       `@font-face { font-family: 'Custom'; src: url('font.woff2'); }`,
			wantRules: 1,
			checkFunc: func(t *testing.T, rules []CSSRule) {
				if rules[0].AtRuleType != "font-face" {
					t.Errorf("expected font-face, got %q", rules[0].AtRuleType)
				}
			},
		},
		{
			name:      "import",
			css:       `@import url('other.css');`,
			wantRules: 1,
			checkFunc: func(t *testing.T, rules []CSSRule) {
				if rules[0].AtRuleType != "import" {
					t.Errorf("expected import, got %q", rules[0].AtRuleType)
				}
			},
		},
		{
			name:      "with comments",
			css:       `/* comment */ .foo { color: red; } /* another comment */`,
			wantRules: 1,
		},
		{
			name:      "nested at-rules",
			css:       `@media screen { @media (min-width: 600px) { .large { width: 100%; } } }`,
			wantRules: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := ParseCSS(tt.css)
			if len(rules) != tt.wantRules {
				t.Errorf("ParseCSS() returned %d rules, want %d", len(rules), tt.wantRules)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, rules)
			}
		})
	}
}

func TestExtractSelectorsFromRule(t *testing.T) {
	tests := []struct {
		selector string
		want     []string
	}{
		{"h1", []string{"h1"}},
		{"h1, h2, h3", []string{"h1", "h2", "h3"}},
		{".foo, .bar", []string{".foo", ".bar"}},
		{".foo,.bar", []string{".foo", ".bar"}},
		{" h1 ,  h2 ", []string{"h1", "h2"}},
	}

	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			got := ExtractSelectorsFromRule(tt.selector)
			if len(got) != len(tt.want) {
				t.Errorf("got %d selectors, want %d", len(got), len(tt.want))
			}
			for i, s := range got {
				if s != tt.want[i] {
					t.Errorf("selector[%d] = %q, want %q", i, s, tt.want[i])
				}
			}
		})
	}
}

func TestExtractClassesFromSelector(t *testing.T) {
	tests := []struct {
		selector string
		want     []string
	}{
		{".foo", []string{"foo"}},
		{".foo.bar", []string{"foo", "bar"}},
		{"div.container", []string{"container"}},
		{".foo .bar", []string{"foo", "bar"}},
		{".foo > .bar", []string{"foo", "bar"}},
		{"#id.class", []string{"class"}},
		{"[type].form-control", []string{"form-control"}},
		{".btn-primary:hover", []string{"btn-primary"}},
	}

	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			got := ExtractClassesFromSelector(tt.selector)
			if len(got) != len(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
				return
			}
			for i, c := range got {
				if c != tt.want[i] {
					t.Errorf("class[%d] = %q, want %q", i, c, tt.want[i])
				}
			}
		})
	}
}

func TestExtractIDsFromSelector(t *testing.T) {
	tests := []struct {
		selector string
		want     []string
	}{
		{"#foo", []string{"foo"}},
		{"#foo#bar", []string{"foo", "bar"}},
		{"div#container", []string{"container"}},
		{"#header .nav", []string{"header"}},
		{".class#id", []string{"id"}},
	}

	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			got := ExtractIDsFromSelector(tt.selector)
			if len(got) != len(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
				return
			}
			for i, id := range got {
				if id != tt.want[i] {
					t.Errorf("ID[%d] = %q, want %q", i, id, tt.want[i])
				}
			}
		})
	}
}

func TestExtractElementsFromSelector(t *testing.T) {
	tests := []struct {
		selector string
		want     []string
	}{
		{"div", []string{"div"}},
		{"div p", []string{"div", "p"}},
		{"div > p", []string{"div", "p"}},
		{"ul li a", []string{"ul", "li", "a"}},
		{".class", []string{}},
		{"div.class", []string{"div"}},
	}

	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			got := ExtractElementsFromSelector(tt.selector)
			if len(got) != len(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
				return
			}
			for i, elem := range got {
				if elem != tt.want[i] {
					t.Errorf("element[%d] = %q, want %q", i, elem, tt.want[i])
				}
			}
		})
	}
}

func TestRemoveComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no comments", ".foo { color: red; }", ".foo { color: red; }"},
		{"single comment", "/* comment */ .foo { color: red; }", " .foo { color: red; }"},
		{"multiple comments", "/* a */ .foo /* b */ { color: red; } /* c */", " .foo  { color: red; } "},
		{"multiline comment", "/*\nmulti\nline\n*/ .foo {}", " .foo {}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeComments(tt.input)
			if got != tt.want {
				t.Errorf("removeComments() = %q, want %q", got, tt.want)
			}
		})
	}
}
