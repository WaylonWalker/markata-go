package serveopen

import (
	"reflect"
	"testing"
)

func TestEditorCommand_UsesLocation(t *testing.T) {
	t.Setenv("EDITOR", "code --wait")
	cmd, err := EditorCommand("posts/page.md", 17)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"code", "--wait", "--goto", "posts/page.md:17"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("args = %q, want %q", cmd.Args, want)
	}
	t.Setenv("EDITOR", "nvim")
	cmd, err = EditorCommand("posts/page.md", 17)
	if err != nil {
		t.Fatal(err)
	}
	if !IsTerminalEditor(cmd) || !reflect.DeepEqual(cmd.Args, []string{"nvim", "+17", "posts/page.md"}) {
		t.Fatalf("terminal editor args = %q", cmd.Args)
	}
}

func TestBrowserCommand_StaysLocal(t *testing.T) {
	t.Setenv("BROWSER", "firefox --new-tab")
	for _, raw := range []string{
		"http://localhost:8000/post/",
		"http://127.0.0.1:8000/post/",
		"http://[::1]:8000/post/",
	} {
		cmd, err := BrowserCommand(raw)
		if err != nil {
			t.Fatalf("BrowserCommand(%q): %v", raw, err)
		}
		if !reflect.DeepEqual(cmd.Args, []string{"firefox", "--new-tab", raw}) {
			t.Fatalf("browser args = %q", cmd.Args)
		}
	}
	for _, raw := range []string{
		"https://localhost:8000/post/",
		"http://example.com/",
		"http://localhost.evil.example/",
		"http://localhost@evil.example/",
		"http://evil.example@localhost:8000/",
		"http://localhost:bad/",
		"http://[::1",
	} {
		if _, err := BrowserCommand(raw); err == nil {
			t.Errorf("BrowserCommand(%q) accepted unsafe URL", raw)
		}
	}
}
