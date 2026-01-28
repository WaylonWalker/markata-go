package models

import (
	"testing"
	"time"
)

// =============================================================================
// SteamConfig Tests
// =============================================================================

func TestNewSteamConfig(t *testing.T) {
	config := NewSteamConfig()

	if config.PostsDir != "pages/steam" {
		t.Errorf("PostsDir: got %q, want %q", config.PostsDir, "pages/steam")
	}
	if config.Template != "steam_achievement" {
		t.Errorf("Template: got %q, want %q", config.Template, "steam_achievement")
	}
	if config.CacheDuration != 3600 {
		t.Errorf("CacheDuration: got %d, want %d", config.CacheDuration, 3600)
	}
	if config.MinPlaytimeHours != 3.0 {
		t.Errorf("MinPlaytimeHours: got %f, want %f", config.MinPlaytimeHours, 3.0)
	}
	if config.Enabled {
		t.Error("Enabled: should be false by default")
	}
}

// =============================================================================
// SteamAchievement Tests
// =============================================================================

func TestSteamAchievement_IsUnlocked(t *testing.T) {
	tests := []struct {
		name     string
		achieved int
		want     bool
	}{
		{
			name:     "unlocked achievement",
			achieved: 1,
			want:     true,
		},
		{
			name:     "locked achievement",
			achieved: 0,
			want:     false,
		},
		{
			name:     "negative value treated as locked",
			achieved: -1,
			want:     false,
		},
		{
			name:     "value greater than 1 treated as locked",
			achieved: 2,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &SteamAchievement{Achieved: tt.achieved}
			if got := sa.IsUnlocked(); got != tt.want {
				t.Errorf("IsUnlocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSteamAchievement_UnlockDate(t *testing.T) {
	unlockTime := int64(1704067200) // 2024-01-01 00:00:00 UTC

	tests := []struct {
		name       string
		achieved   int
		unlockTime *int64
		wantNil    bool
	}{
		{
			name:       "unlocked with timestamp",
			achieved:   1,
			unlockTime: &unlockTime,
			wantNil:    false,
		},
		{
			name:       "locked achievement returns nil",
			achieved:   0,
			unlockTime: &unlockTime,
			wantNil:    true,
		},
		{
			name:       "unlocked but nil timestamp returns nil",
			achieved:   1,
			unlockTime: nil,
			wantNil:    true,
		},
		{
			name:       "locked with nil timestamp returns nil",
			achieved:   0,
			unlockTime: nil,
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &SteamAchievement{
				Achieved:   tt.achieved,
				UnlockTime: tt.unlockTime,
			}
			got := sa.UnlockDate()
			if tt.wantNil {
				if got != nil {
					t.Errorf("UnlockDate() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Error("UnlockDate() = nil, want non-nil")
				} else {
					expected := time.Unix(unlockTime, 0)
					if !got.Equal(expected) {
						t.Errorf("UnlockDate() = %v, want %v", got, expected)
					}
				}
			}
		})
	}
}

// =============================================================================
// SteamGame Tests
// =============================================================================

func TestSteamGame_PlaytimeHours(t *testing.T) {
	tests := []struct {
		name            string
		playtimeForever *int
		want            float64
	}{
		{
			name:            "nil playtime returns 0",
			playtimeForever: nil,
			want:            0,
		},
		{
			name:            "zero minutes returns 0",
			playtimeForever: intPtr(0),
			want:            0,
		},
		{
			name:            "60 minutes returns 1 hour",
			playtimeForever: intPtr(60),
			want:            1.0,
		},
		{
			name:            "90 minutes returns 1.5 hours",
			playtimeForever: intPtr(90),
			want:            1.5,
		},
		{
			name:            "180 minutes returns 3 hours",
			playtimeForever: intPtr(180),
			want:            3.0,
		},
		{
			name:            "30 minutes returns 0.5 hours",
			playtimeForever: intPtr(30),
			want:            0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sg := &SteamGame{PlaytimeForever: tt.playtimeForever}
			if got := sg.PlaytimeHours(); got != tt.want {
				t.Errorf("PlaytimeHours() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSteamGame_LastPlayedDate(t *testing.T) {
	lastPlayed := int64(1704067200) // 2024-01-01 00:00:00 UTC

	tests := []struct {
		name       string
		lastPlayed *int64
		wantNil    bool
	}{
		{
			name:       "nil last played returns nil",
			lastPlayed: nil,
			wantNil:    true,
		},
		{
			name:       "valid timestamp returns time",
			lastPlayed: &lastPlayed,
			wantNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sg := &SteamGame{LastPlayed: tt.lastPlayed}
			got := sg.LastPlayedDate()
			if tt.wantNil {
				if got != nil {
					t.Errorf("LastPlayedDate() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Error("LastPlayedDate() = nil, want non-nil")
				} else {
					expected := time.Unix(lastPlayed, 0)
					if !got.Equal(expected) {
						t.Errorf("LastPlayedDate() = %v, want %v", got, expected)
					}
				}
			}
		})
	}
}

func TestSteamGame_ShouldInclude(t *testing.T) {
	tests := []struct {
		name            string
		playtimeForever *int
		minHours        float64
		want            bool
	}{
		{
			name:            "nil playtime with positive min hours",
			playtimeForever: nil,
			minHours:        1.0,
			want:            false,
		},
		{
			name:            "nil playtime with zero min hours",
			playtimeForever: nil,
			minHours:        0,
			want:            true,
		},
		{
			name:            "playtime equals min hours",
			playtimeForever: intPtr(180), // 3 hours
			minHours:        3.0,
			want:            true,
		},
		{
			name:            "playtime exceeds min hours",
			playtimeForever: intPtr(240), // 4 hours
			minHours:        3.0,
			want:            true,
		},
		{
			name:            "playtime below min hours",
			playtimeForever: intPtr(120), // 2 hours
			minHours:        3.0,
			want:            false,
		},
		{
			name:            "zero min hours always includes",
			playtimeForever: intPtr(1),
			minHours:        0,
			want:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sg := &SteamGame{PlaytimeForever: tt.playtimeForever}
			if got := sg.ShouldInclude(tt.minHours); got != tt.want {
				t.Errorf("ShouldInclude(%v) = %v, want %v", tt.minHours, got, tt.want)
			}
		})
	}
}

func TestSteamGame_UnlockedAchievementsList(t *testing.T) {
	tests := []struct {
		name         string
		achievements []SteamAchievement
		wantCount    int
	}{
		{
			name:         "empty achievements",
			achievements: []SteamAchievement{},
			wantCount:    0,
		},
		{
			name: "all locked",
			achievements: []SteamAchievement{
				{APIName: "ach1", Achieved: 0},
				{APIName: "ach2", Achieved: 0},
			},
			wantCount: 0,
		},
		{
			name: "all unlocked",
			achievements: []SteamAchievement{
				{APIName: "ach1", Achieved: 1},
				{APIName: "ach2", Achieved: 1},
			},
			wantCount: 2,
		},
		{
			name: "mixed locked and unlocked",
			achievements: []SteamAchievement{
				{APIName: "ach1", Achieved: 1},
				{APIName: "ach2", Achieved: 0},
				{APIName: "ach3", Achieved: 1},
				{APIName: "ach4", Achieved: 0},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sg := &SteamGame{Achievements: tt.achievements}
			got := sg.UnlockedAchievementsList()
			if len(got) != tt.wantCount {
				t.Errorf("UnlockedAchievementsList() returned %d items, want %d", len(got), tt.wantCount)
			}
			// Verify all returned items are actually unlocked
			for _, ach := range got {
				if !ach.IsUnlocked() {
					t.Errorf("UnlockedAchievementsList() returned locked achievement: %s", ach.APIName)
				}
			}
		})
	}
}

func TestSteamGame_LockedAchievementsList(t *testing.T) {
	tests := []struct {
		name         string
		achievements []SteamAchievement
		wantCount    int
	}{
		{
			name:         "empty achievements",
			achievements: []SteamAchievement{},
			wantCount:    0,
		},
		{
			name: "all unlocked",
			achievements: []SteamAchievement{
				{APIName: "ach1", Achieved: 1},
				{APIName: "ach2", Achieved: 1},
			},
			wantCount: 0,
		},
		{
			name: "all locked",
			achievements: []SteamAchievement{
				{APIName: "ach1", Achieved: 0},
				{APIName: "ach2", Achieved: 0},
			},
			wantCount: 2,
		},
		{
			name: "mixed locked and unlocked",
			achievements: []SteamAchievement{
				{APIName: "ach1", Achieved: 1},
				{APIName: "ach2", Achieved: 0},
				{APIName: "ach3", Achieved: 1},
				{APIName: "ach4", Achieved: 0},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sg := &SteamGame{Achievements: tt.achievements}
			got := sg.LockedAchievementsList()
			if len(got) != tt.wantCount {
				t.Errorf("LockedAchievementsList() returned %d items, want %d", len(got), tt.wantCount)
			}
			// Verify all returned items are actually locked
			for _, ach := range got {
				if ach.IsUnlocked() {
					t.Errorf("LockedAchievementsList() returned unlocked achievement: %s", ach.APIName)
				}
			}
		})
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func intPtr(i int) *int {
	return &i
}
