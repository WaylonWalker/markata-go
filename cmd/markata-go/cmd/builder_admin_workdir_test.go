package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateBuilderAdminWorkDir(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	siteDir := filepath.Join(sourceDir, "public")
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
		{name: "inside source outside release root", workDir: filepath.Join(sourceDir, ".work"), wantErr: true},
		{name: "release root", workDir: siteDir, wantErr: true},
		{name: "legacy workspace under release root", workDir: filepath.Join(siteDir, ".build-work"), wantErr: false},
		{name: "custom workspace under release root", workDir: filepath.Join(siteDir, ".build-work-fast"), wantErr: false},
		{name: "releases directory", workDir: filepath.Join(siteDir, "releases"), wantErr: true},
		{name: "release child", workDir: filepath.Join(siteDir, "releases", "candidate"), wantErr: true},
		{name: "current directory", workDir: filepath.Join(siteDir, "current"), wantErr: true},
		{name: "current child", workDir: filepath.Join(siteDir, "current", "tmp"), wantErr: true},
		{name: "builder admin history", workDir: filepath.Join(siteDir, ".builder-admin"), wantErr: true},
		{name: "builder admin history child", workDir: filepath.Join(siteDir, ".builder-admin", "runs"), wantErr: true},
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
