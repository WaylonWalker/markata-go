package plugins

import (
	"strings"
	"testing"
)

func TestFetchTailwindChecksum_ParsesDotSlashEntries(t *testing.T) {
	content := "4af3198c015616ea7d6617974ec3d70d987ecc00c1ca8463b0a30fd65cc7c06e  ./tailwindcss-linux-x64\n"
	targetName := "tailwindcss-linux-x64"

	var got string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		filename := strings.TrimPrefix(fields[1], "./")
		if filename == targetName {
			got = fields[0]
			break
		}
	}

	if got != "4af3198c015616ea7d6617974ec3d70d987ecc00c1ca8463b0a30fd65cc7c06e" {
		t.Fatalf("checksum = %q, want expected value", got)
	}
}
