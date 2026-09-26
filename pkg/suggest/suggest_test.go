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
