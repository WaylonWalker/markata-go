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
	cmd, err := BrowserCommand("http://localhost:8000/post/")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cmd.Args, []string{"firefox", "--new-tab", "http://localhost:8000/post/"}) {
		t.Fatalf("browser args = %q", cmd.Args)
	}
	if _, err := BrowserCommand("https://example.com"); err == nil {
		t.Fatal("expected nonlocal URL rejection")
	}
}
