package models

import (
	"time"
)

// SteamConfig represents configuration for the Steam achievements plugin.
type SteamConfig struct {
	// APIKey is the Steam Web API key (required)
	APIKey string `json:"api_key" yaml:"api_key" toml:"api_key" env:"STEAM_API_KEY"`

	// SteamID is the user's Steam ID (required)
	SteamID string `json:"steam_id" yaml:"steam_id" toml:"steam_id" env:"STEAM_ID"`

	// PostsDir is the directory to create achievement posts (default: "pages/steam")
	PostsDir string `json:"posts_dir,omitempty" yaml:"posts_dir,omitempty" toml:"posts_dir,omitempty"`

	// Template is the template name to use for steam posts (default: "steam_achievement")
	Template string `json:"template,omitempty" yaml:"template,omitempty" toml:"template,omitempty"`

	// CacheDuration is cache duration in seconds (default: 3600)
	CacheDuration int `json:"cache_duration,omitempty" yaml:"cache_duration,omitempty" toml:"cache_duration,omitempty"`

	// MinPlaytimeHours is minimum playtime in hours to include games (default: 3)
	MinPlaytimeHours float64 `json:"min_playtime_hours,omitempty" yaml:"min_playtime_hours,omitempty" toml:"min_playtime_hours,omitempty"`

	// Enabled enables the plugin (default: false)
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty" toml:"enabled,omitempty"`
}

// NewSteamConfig returns a new SteamConfig with default values.
func NewSteamConfig() SteamConfig {
	return SteamConfig{
		PostsDir:         "pages/steam",
		Template:         "steam_achievement",
		CacheDuration:    3600,
		MinPlaytimeHours: 3.0,
		Enabled:          false,
	}
}

// SteamGame represents a Steam game with achievement data.
type SteamGame struct {
	// AppID is the Steam application ID
	AppID int `json:"appid" yaml:"appid" toml:"appid"`

	// Name is the game name
	Name string `json:"name" yaml:"name" toml:"name"`

	// Description is the game description
	Description *string `json:"description,omitempty" yaml:"description,omitempty" toml:"description,omitempty"`

	// Developers are the game developers
	Developers []string `json:"developers,omitempty" yaml:"developers,omitempty" toml:"developers,omitempty"`

	// Publishers are the game publishers
	Publishers []string `json:"publishers,omitempty" yaml:"publishers,omitempty" toml:"publishers,omitempty"`

	// Achievements is the list of achievements
	Achievements []SteamAchievement `json:"achievements" yaml:"achievements" toml:"achievements"`

	// TotalAchievements is the total number of achievements
	TotalAchievements int `json:"total_achievements" yaml:"total_achievements" toml:"total_achievements"`

	// UnlockedAchievements is the number of unlocked achievements
	UnlockedAchievements int `json:"unlocked_achievements" yaml:"unlocked_achievements" toml:"unlocked_achievements"`

	// CompletionPercentage is the percentage of achievements completed
	CompletionPercentage float64 `json:"completion_percentage" yaml:"completion_percentage" toml:"completion_percentage"`

	// PlaytimeForever is total playtime in minutes
	PlaytimeForever *int `json:"playtime_forever,omitempty" yaml:"playtime_forever,omitempty" toml:"playtime_forever,omitempty"`

	// Playtime2Weeks is playtime in the last 2 weeks in minutes
	Playtime2Weeks *int `json:"playtime_2weeks,omitempty" yaml:"playtime_2weeks,omitempty" toml:"playtime_2weeks,omitempty"`

	// LastPlayed is the timestamp when the game was last played
	LastPlayed *int64 `json:"last_played,omitempty" yaml:"last_played,omitempty" toml:"last_played,omitempty"`
}

// SteamAchievement represents a Steam achievement.
type SteamAchievement struct {
	// APIName is the internal achievement API name
	APIName string `json:"apiname" yaml:"apiname" toml:"apiname"`

	// Name is the display name of the achievement
	Name *string `json:"name,omitempty" yaml:"name,omitempty" toml:"name,omitempty"`

	// Description is the achievement description
	Description *string `json:"description,omitempty" yaml:"description,omitempty" toml:"description,omitempty"`

	// Icon is the URL to the achievement icon
	Icon *string `json:"icon,omitempty" yaml:"icon,omitempty" toml:"icon,omitempty"`

	// IconGray is the URL to the grayed out achievement icon
	IconGray *string `json:"icongray,omitempty" yaml:"icongray,omitempty" toml:"icongray,omitempty"`

	// Achieved indicates if the achievement is unlocked (1 or 0)
	Achieved int `json:"achieved" yaml:"achieved" toml:"achieved"`

	// UnlockTime is the timestamp when the achievement was unlocked
	UnlockTime *int64 `json:"unlocktime,omitempty" yaml:"unlocktime,omitempty" toml:"unlocktime,omitempty"`
}

// IsUnlocked returns true if the achievement is unlocked.
func (sa *SteamAchievement) IsUnlocked() bool {
	return sa.Achieved == 1
}

// UnlockDate returns the unlock date as a time.Time, or nil if not unlocked.
func (sa *SteamAchievement) UnlockDate() *time.Time {
	if !sa.IsUnlocked() || sa.UnlockTime == nil {
		return nil
	}
	unlockTime := time.Unix(*sa.UnlockTime, 0)
	return &unlockTime
}

// PlaytimeHours returns the total playtime in hours.
func (sg *SteamGame) PlaytimeHours() float64 {
	if sg.PlaytimeForever == nil {
		return 0
	}
	return float64(*sg.PlaytimeForever) / 60.0
}

// LastPlayedDate returns the last played date as a time.Time, or nil if never played.
func (sg *SteamGame) LastPlayedDate() *time.Time {
	if sg.LastPlayed == nil {
		return nil
	}
	lastPlayed := time.Unix(*sg.LastPlayed, 0)
	return &lastPlayed
}

// ShouldInclude returns true if the game meets the minimum playtime requirement.
func (sg *SteamGame) ShouldInclude(minHours float64) bool {
	return sg.PlaytimeHours() >= minHours
}

// UnlockedAchievementsList returns a list of unlocked achievements.
func (sg *SteamGame) UnlockedAchievementsList() []SteamAchievement {
	var unlocked []SteamAchievement
	for _, achievement := range sg.Achievements {
		if achievement.IsUnlocked() {
			unlocked = append(unlocked, achievement)
		}
	}
	return unlocked
}

// LockedAchievementsList returns a list of locked achievements.
func (sg *SteamGame) LockedAchievementsList() []SteamAchievement {
	var locked []SteamAchievement
	for _, achievement := range sg.Achievements {
		if !achievement.IsUnlocked() {
			locked = append(locked, achievement)
		}
	}
	return locked
}
