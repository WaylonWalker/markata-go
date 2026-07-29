package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/builderadmin"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestApplyBuilderAdminConfigHeaders(t *testing.T) {
	userID := "X-Authentik-Uid"
	empty := ""
	headers := builderadmin.DefaultAuthHeaders()
	applyBuilderAdminConfigHeaders(&headers, models.BuilderAdminAuthHeadersConfig{
		UserID:      &userID,
		DisplayName: &empty,
	})
	if headers.UserID != userID {
		t.Fatalf("UserID = %q, want %q", headers.UserID, userID)
	}
	if headers.DisplayName != "" {
		t.Fatalf("DisplayName = %q, want empty", headers.DisplayName)
	}
	if headers.Username != "X-Hlab-Username" {
		t.Fatalf("Username = %q, want default", headers.Username)
	}
}

func TestResolveBuilderAdminAuthHeaders_ConfigFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "markata-go.toml")
	if err := os.WriteFile(configPath, []byte(`
[markata-go.builder_admin.auth.headers]
user_id = "X-Authentik-Uid"
display_name = ""
`), 0o600); err != nil {
		t.Fatal(err)
	}
	headers, err := resolveBuilderAdminAuthHeaders(builderAdminCmd, configPath)
	if err != nil {
		t.Fatal(err)
	}
	if headers.UserID != "X-Authentik-Uid" || headers.DisplayName != "" {
		t.Fatalf("headers = %#v", headers)
	}
}

func TestApplyBuilderAdminExtraHeaders(t *testing.T) {
	headers := builderadmin.DefaultAuthHeaders()
	applyBuilderAdminExtraHeaders(&headers, map[string]any{
		"builder_admin": map[string]any{
			"auth": map[string]any{
				"headers": map[string]any{
					"user_id":      "X-Authentik-Uid",
					"display_name": "",
				},
			},
		},
	})
	if headers.UserID != "X-Authentik-Uid" || headers.DisplayName != "" {
		t.Fatalf("headers = %#v", headers)
	}
}

func TestResolveBuilderAdminConfigPath(t *testing.T) {
	sourceDir := t.TempDir()
	configPath := filepath.Join(sourceDir, "markata-go.toml")
	if err := os.WriteFile(configPath, []byte("[markata-go]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := resolveBuilderAdminConfigPath("", sourceDir); got != configPath {
		t.Fatalf("resolveBuilderAdminConfigPath() = %q, want %q", got, configPath)
	}
	if got := resolveBuilderAdminConfigPath("config/site.toml", sourceDir); got != filepath.Join(sourceDir, "config", "site.toml") {
		t.Fatalf("relative config path = %q", got)
	}
}
