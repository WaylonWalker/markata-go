package agentskill

import (
	"io/fs"
	"testing"
)

func TestSiteSkill_ReturnsValidFS(t *testing.T) {
	skillFS, err := SiteSkill()
	if err != nil {
		t.Fatalf("SiteSkill() error = %v", err)
	}

	f, err := skillFS.Open("SKILL.md")
	if err != nil {
		t.Fatalf("expected SKILL.md in bundled skill FS: %v", err)
	}
	f.Close()
}

func TestSiteSkill_ContainsAllRequiredTopics(t *testing.T) {
	requiredTopics := []string{
		"topics/configuration.md",
		"topics/writing-frontmatter.md",
		"topics/cli-usage.md",
		"topics/build-deployment.md",
		"topics/faster-builds.md",
		"topics/theme-creation.md",
		"topics/template-management.md",
		"topics/plugin-creation.md",
	}

	skillFS, err := SiteSkill()
	if err != nil {
		t.Fatalf("SiteSkill() error = %v", err)
	}

	for _, topic := range requiredTopics {
		t.Run(topic, func(t *testing.T) {
			info, err := fs.Stat(skillFS, topic)
			if err != nil {
				t.Fatalf("missing required topic file %q: %v", topic, err)
			}
			if info.Size() == 0 {
				t.Fatalf("topic file %q is empty", topic)
			}
		})
	}
}

func TestListFiles_ReturnsExpectedFiles(t *testing.T) {
	files, err := ListFiles()
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}

	// SKILL.md + 8 topic files = 9 minimum
	if len(files) < 9 {
		t.Fatalf("ListFiles() returned %d files, expected at least 9", len(files))
	}

	hasSkill := false
	for _, f := range files {
		if f == "SKILL.md" {
			hasSkill = true
			break
		}
	}
	if !hasSkill {
		t.Fatal("ListFiles() did not include SKILL.md")
	}
}

func TestListFiles_IsSorted(t *testing.T) {
	files, err := ListFiles()
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}

	for i := 1; i < len(files); i++ {
		if files[i] < files[i-1] {
			t.Fatalf("ListFiles() not sorted: %q came after %q", files[i], files[i-1])
		}
	}
}

func TestSiteSkillName_IsExpectedValue(t *testing.T) {
	if SiteSkillName != "markata-go-site" {
		t.Fatalf("SiteSkillName = %q, want %q", SiteSkillName, "markata-go-site")
	}
}
