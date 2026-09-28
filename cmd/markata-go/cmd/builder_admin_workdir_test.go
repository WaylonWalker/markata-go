package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateBuilderAdminWorkDir(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	siteDir := filepath.Join(root, "site")
	safeDir := filepath.Join(root, "work", "build")
	filesystemRoot := filepath.VolumeName(root) + string(os.PathSeparator)

	tests := []struct {
		name    string
		workDir string
		wantErr bool
	}{
		{name: "default empty", workDir: "", wantErr: false},
		{name: "separate workspace", workDir: safeDir, wantErr: false},
		{name: "filesystem root", workDir: filesystemRoot, wantErr: true},
		{name: "source root", workDir: sourceDir, wantErr: true},
		{name: "inside source", workDir: filepath.Join(sourceDir, ".work"), wantErr: true},
		{name: "release root", workDir: siteDir, wantErr: true},
		{name: "legacy workspace under release root", workDir: filepath.Join(siteDir, ".build-work"), wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBuilderAdminWorkDir(tt.workDir, sourceDir, siteDir)
			if tt.wantErr && err == nil {
				t.Fatalf("validateBuilderAdminWorkDir(%q) succeeded, want error", tt.workDir)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateBuilderAdminWorkDir(%q) error = %v", tt.workDir, err)
			}
		})
	}
}
