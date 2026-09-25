package plugins

import (
	"encoding/json"
	"fmt"
	"time"
)

// seasonalLeadDays is how many days before a holiday its palette appears.
const seasonalLeadDays = 3

// seasonalYearsAhead bounds how far ahead computed holiday dates are listed.
const seasonalYearsAhead = 10

type seasonalSeason struct {
	label    string
	start    int // MMDD the season begins (northern hemisphere, astronomical)
	palettes []string
}

type seasonalHoliday struct {
	label    string
	fixed    string   // "MM-DD" for holidays on the same date every year
	dates    []string // "YYYY-MM-DD" for holidays that move each year
	easter   bool     // dates computed with the Gregorian computus
	days     int      // length of the celebration (default 1)
	palettes []string // palette ids; the first match per color mode wins
}

// Northern hemisphere seasons, ordered by start date. Winter wraps the year.
var seasonalSeasons = []seasonalSeason{
	{label: "Spring", start: 320, palettes: []string{"pollen8"}},
	{label: "Summer", start: 621, palettes: []string{"summer-beach"}},
	{label: "Autumn", start: 922, palettes: []string{"autumn"}},
	{label: "Winter", start: 1221, palettes: []string{"winter-frost"}},
}

// World holidays with a matching built-in palette. Moving dates are the main
// day of the festival (Lakshmi Puja for Diwali, first full day for Hanukkah).
var seasonalHolidays = []seasonalHoliday{
	{label: "New Year", fixed: "01-01", palettes: []string{"white-gold", "black-gold"}},
	{label: "Lunar New Year", palettes: []string{"lunar-new-year"}, dates: []string{
		"2025-01-29", "2026-02-17", "2027-02-06", "2028-01-26", "2029-02-13", "2030-02-03",
		"2031-01-23", "2032-02-11", "2033-01-31", "2034-02-19", "2035-02-08", "2036-01-28",
	}},
	{label: "Valentine's Day", fixed: "02-14", palettes: []string{"valentine"}},
	{label: "St. Patrick's Day", fixed: "03-17", palettes: []string{"st-patricks", "st.-patrick's-day"}},
	{label: "Easter", easter: true, palettes: []string{"blessing"}},
	{label: "Earth Day", fixed: "04-22", palettes: []string{"everforest-light", "everforest-dark"}},
	{label: "Halloween", fixed: "10-31", palettes: []string{"halloween"}},
	{label: "Diwali", palettes: []string{"diwali"}, dates: []string{
		"2025-10-20", "2026-11-08", "2027-10-29", "2028-10-17", "2029-11-05", "2030-10-26",
		"2031-11-14", "2032-11-02", "2033-10-22",
	}},
	{label: "Hanukkah", days: 8, palettes: []string{"hanukkah"}, dates: []string{
		"2025-12-15", "2026-12-05", "2027-12-25", "2028-12-13", "2029-12-03", "2030-12-21",
		"2031-12-11", "2032-11-29", "2033-12-17", "2034-12-07", "2035-12-26",
	}},
	{label: "Christmas", fixed: "12-25", palettes: []string{"christmas"}},
}

// seasonalEntry is the compact JSON shape read by the head script in base.html.
type seasonalEntry struct {
	Label string   `json:"l"`
	Start int      `json:"f,omitempty"`
	Dates []string `json:"d,omitempty"`
	Days  int      `json:"n,omitempty"`
	Light string   `json:"lt"`
	Dark  string   `json:"dk"`
}

type seasonalSchedule struct {
	Lead     int             `json:"lead"`
	Seasons  []seasonalEntry `json:"s"`
	Holidays []seasonalEntry `json:"h"`
}

// easterDate returns Western Easter Sunday for year (anonymous Gregorian algorithm).
func easterDate(year int) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// resolveSeasonalPair picks the light and dark manifest entries for a list of
// palette ids, falling back to the counterpart of the first id that exists.
func resolveSeasonalPair(index map[string]PaletteManifestEntry, ids []string) (light, dark string) {
	var first *PaletteManifestEntry
	for _, id := range ids {
		entry, ok := index[id]
		if !ok {
			continue
		}
		if first == nil {
			e := entry
			first = &e
		}
		if entry.Variant == themeModeLight && light == "" {
			light = entry.Name
		}
		if entry.Variant == themeModeDark && dark == "" {
			dark = entry.Name
		}
	}
	if first == nil {
		return "", ""
	}
	if light == "" && first.Variant == themeModeDark {
		if _, ok := index[first.Counterpart]; ok {
			light = first.Counterpart
		}
	}
	if dark == "" && first.Variant == themeModeLight {
		if _, ok := index[first.Counterpart]; ok {
			dark = first.Counterpart
		}
	}
	return light, dark
}

// buildSeasonalSchedule resolves the seasonal calendar against the palettes
// the picker ships. Entries whose palettes were filtered out are dropped; it
// returns nil when no season can be resolved.
func buildSeasonalSchedule(manifest []PaletteManifestEntry, now time.Time) *seasonalSchedule {
	index := make(map[string]PaletteManifestEntry, len(manifest))
	for _, entry := range manifest {
		index[entry.Name] = entry
	}
	schedule := &seasonalSchedule{Lead: seasonalLeadDays}
	for _, season := range seasonalSeasons {
		light, dark := resolveSeasonalPair(index, season.palettes)
		if light == "" || dark == "" {
			continue
		}
		schedule.Seasons = append(schedule.Seasons, seasonalEntry{Label: season.label, Start: season.start, Light: light, Dark: dark})
	}
	if len(schedule.Seasons) == 0 {
		return nil
	}
	thisYear := now.Year()
	for _, holiday := range seasonalHolidays {
		light, dark := resolveSeasonalPair(index, holiday.palettes)
		if light == "" || dark == "" {
			continue
		}
		var dates []string
		switch {
		case holiday.fixed != "":
			dates = []string{holiday.fixed}
		case holiday.easter:
			for year := thisYear - 1; year <= thisYear+seasonalYearsAhead; year++ {
				dates = append(dates, easterDate(year).Format("2006-01-02"))
			}
		default:
			for _, date := range holiday.dates {
				// Keep the list short: drop dates that ended before last year.
				if len(date) >= 4 && date[:4] >= fmt.Sprintf("%04d", thisYear-1) {
					dates = append(dates, date)
				}
			}
		}
		if len(dates) == 0 {
			continue
		}
		days := holiday.days
		if days <= 1 {
			days = 0
		}
		schedule.Holidays = append(schedule.Holidays, seasonalEntry{Label: holiday.label, Dates: dates, Days: days, Light: light, Dark: dark})
	}
	return schedule
}

// seasonalScheduleJSON renders the schedule for the base.html head script.
// json.Marshal escapes <, >, and & so the result is safe inside <script>.
func seasonalScheduleJSON(manifest []PaletteManifestEntry, now time.Time) string {
	schedule := buildSeasonalSchedule(manifest, now)
	if schedule == nil {
		return ""
	}
	data, err := json.Marshal(schedule)
	if err != nil {
		return ""
	}
	return string(data)
}
