package templates

import "testing"

func TestMediaDimensionsFromURL(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantWidth  int
		wantHeight int
		wantOK     bool
	}{
		{name: "w and h", input: "https://example.com/image.jpg?w=1200&h=675", wantWidth: 1200, wantHeight: 675, wantOK: true},
		{name: "width and height", input: "https://example.com/image.jpg?width=800&height=600", wantWidth: 800, wantHeight: 600, wantOK: true},
		{name: "width only", input: "https://example.com/image.jpg?w=900", wantWidth: 900, wantHeight: 0, wantOK: true},
		{name: "unrelated query", input: "https://example.com/image.jpg?foo=bar", wantWidth: 0, wantHeight: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height, ok := MediaDimensionsFromURL(tt.input)
			if width != tt.wantWidth || height != tt.wantHeight || ok != tt.wantOK {
				t.Fatalf("MediaDimensionsFromURL(%q) = (%d, %d, %v), want (%d, %d, %v)", tt.input, width, height, ok, tt.wantWidth, tt.wantHeight, tt.wantOK)
			}
		})
	}
}
