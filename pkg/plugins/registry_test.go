package plugins

import "testing"

func TestDefaultPlugins_StatsRegisteredOnce(t *testing.T) {
	count := 0
	for _, plugin := range DefaultPlugins() {
		if plugin.Name() == "stats" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("stats plugin registrations = %d, want 1", count)
	}
}
