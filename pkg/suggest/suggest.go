// Package suggest ranks close names for user-facing diagnostics.
package suggest

import (
	"sort"
	"strings"
)

type match struct {
	name     string
	distance int
	score    int
}

// Family groups executable spellings for ranking while keeping semantic
// suggestions separate from aliases.
type Family struct {
	Name       string
	Aliases    []string
	SuggestFor []string
}

// Ranked returns unique command families, ranked by semantic match, prefix,
// then transposition-aware edit distance.
func Ranked(input string, families []Family, limit int) []string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" || limit <= 0 {
		return nil
	}
	matches := make([]match, 0, len(families))
	for _, family := range families {
		best := 100
		for _, semantic := range family.SuggestFor {
			if strings.EqualFold(input, semantic) {
				best = 0
			}
		}
		for _, name := range append([]string{family.Name}, family.Aliases...) {
			candidate := strings.ToLower(name)
			score := 100
			switch {
			case candidate == input:
				score = 0
			case strings.HasPrefix(candidate, input):
				score = 1
			case strings.HasPrefix(input, candidate):
				score = 2
			default:
				if d := editDistance(input, candidate); d <= 2 {
					score = 3 + d
				}
			}
			best = min(best, score)
		}
		if best < 100 {
			matches = append(matches, match{name: family.Name, score: best})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score < matches[j].score
		}
		return matches[i].name < matches[j].name
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	out := make([]string, len(matches))
	for i, item := range matches {
		out[i] = item.name
	}
	return out
}

// Closest returns up to limit distinct, plausible names in deterministic order.
// It deliberately does not make a choice for the caller.
func Closest(input string, candidates []string, limit int) []string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" || limit <= 0 {
		return nil
	}
	maxDistance := 1
	if len(input) >= 5 {
		maxDistance = 2
	}
	if len(input) >= 9 {
		maxDistance = 3
	}
	seen := make(map[string]bool, len(candidates))
	matches := make([]match, 0, len(candidates))
	for _, candidate := range candidates {
		key := strings.ToLower(candidate)
		if seen[key] || key == input {
			continue
		}
		seen[key] = true
		distance := editDistance(input, key)
		if distance <= maxDistance {
			matches = append(matches, match{name: candidate, distance: distance})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].distance != matches[j].distance {
			return matches[i].distance < matches[j].distance
		}
		return matches[i].name < matches[j].name
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	result := make([]string, len(matches))
	for i, candidate := range matches {
		result[i] = candidate.name
	}
	return result
}

// editDistance is Damerau-Levenshtein distance with adjacent transpositions.
func editDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	rows := make([][]int, len(ar)+1)
	for i := range rows {
		rows[i] = make([]int, len(br)+1)
		rows[i][0] = i
	}
	for j := range rows[0] {
		rows[0][j] = j
	}
	for i := 1; i <= len(ar); i++ {
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			rows[i][j] = min(rows[i-1][j]+1, rows[i][j-1]+1, rows[i-1][j-1]+cost)
			if i > 1 && j > 1 && ar[i-1] == br[j-2] && ar[i-2] == br[j-1] {
				rows[i][j] = min(rows[i][j], rows[i-2][j-2]+1)
			}
		}
	}
	return rows[len(ar)][len(br)]
}
