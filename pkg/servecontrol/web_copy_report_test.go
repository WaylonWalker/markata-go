package servecontrol

import (
	"strings"
	"testing"
)

func TestDashboardIncludesCopyProblemsReport(t *testing.T) {
	required := []string{
		`id="copy-report-button"`,
		`function diagnosticReport(items)`,
		`visibleItems().map(x=>x.item)`,
		`navigator.clipboard.writeText(report)`,
		`Location: `,
		`Explanation: `,
		`Suggested fix: `,
		`Job: `,
		`Step: `,
	}
	for _, fragment := range required {
		if !strings.Contains(webHTML, fragment) {
			t.Fatalf("Serve Control Center is missing copy-report behavior %q", fragment)
		}
	}
}
