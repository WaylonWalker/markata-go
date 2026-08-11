package lsp

import "testing"

func TestGetThemeConfigCompletions_HeadingTextureKind(t *testing.T) {
	content := "[markata-go.theme.heading_texture]\nkind = \"\""
	items := getThemeConfigCompletions(content, 1, `kind = ""`)

	want := map[string]bool{
		"inherit":     true,
		"none":        true,
		"splatter":    true,
		"brush":       true,
		"grunge":      true,
		"screenprint": true,
	}
	if len(items) != len(want) {
		t.Fatalf("got %d heading texture completions, want %d: %#v", len(items), len(want), items)
	}
	for _, item := range items {
		if !want[item.Label] {
			t.Errorf("unexpected heading texture completion %q", item.Label)
		}
		if item.Kind != CompletionItemKindEnumMember {
			t.Errorf("completion %q kind = %v, want enum member", item.Label, item.Kind)
		}
	}
}

func TestGetThemeConfigCompletions_Variables(t *testing.T) {
	items := getThemeConfigCompletions("[markata-go.theme.variables]\n--", 1, "--")
	if len(items) == 0 {
		t.Fatal("expected CSS custom-property completions")
	}
	for _, item := range items {
		if item.Label == "palette" || item.Label == "kind" {
			t.Fatalf("theme key leaked into variables completion: %q", item.Label)
		}
	}
}
