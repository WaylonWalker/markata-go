package suggest

import "testing"

func TestClosest(t *testing.T) {
	tests := []struct {
		input      string
		candidates []string
		want       string
	}{
		{"sevre", []string{"search", "serve"}, "serve"},
		{"hst", []string{"host", "port"}, "host"},
		{"output_dr", []string{"output_dir", "other"}, "output_dir"},
		{"xyzzy", []string{"serve", "build"}, ""},
	}
	for _, test := range tests {
		got := Closest(test.input, test.candidates, 3)
		if test.want == "" && len(got) != 0 {
			t.Errorf("Closest(%q) = %v; want none", test.input, got)
		} else if test.want != "" && (len(got) == 0 || got[0] != test.want) {
			t.Errorf("Closest(%q) = %v; want %q first", test.input, got, test.want)
		}
	}
}

func TestRankedCommandFamilies(t *testing.T) {
	families := []Family{{Name: "serve", Aliases: []string{"s", "serv"}, SuggestFor: []string{"dev", "preview"}}, {Name: "search"}}
	for input, want := range map[string]string{"ser": "serve", "sevre": "serve", "dev": "serve", "preview": "serve"} {
		got := Ranked(input, families, 3)
		if len(got) == 0 || got[0] != want {
			t.Errorf("Ranked(%q) = %v, want %s first", input, got, want)
		}
		for _, name := range got[1:] {
			if name == "serve" {
				t.Errorf("Ranked(%q) duplicated command family: %v", input, got)
			}
		}
	}
	if got := Ranked("xyzzy", families, 3); len(got) != 0 {
		t.Fatalf("gibberish suggestions = %v, want none", got)
	}
}
