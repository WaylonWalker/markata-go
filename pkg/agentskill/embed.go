package agentskill

import (
	"embed"
	"io/fs"
	"path"
	"sort"
)

// SiteSkillName is the default bundled skill name for markata-go sites.
const SiteSkillName = "markata-go-site"

//go:embed all:bundle/markata-go-site
var bundledSkills embed.FS

// SiteSkill returns the embedded filesystem for the bundled markata-go site skill.
func SiteSkill() (fs.FS, error) {
	return fs.Sub(bundledSkills, "bundle/markata-go-site")
}

// ListFiles returns all bundled skill files with slash-separated relative paths.
func ListFiles() ([]string, error) {
	skillFS, err := SiteSkill()
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, 16)
	err = fs.WalkDir(skillFS, ".", func(filePath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path.Clean(filePath))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}
