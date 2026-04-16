// Command gensynonyms parses WordNet data files and generates a curated
// synonym JSON file for embedding in the markata-go binary.
package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Common/frequent English words to prioritize (synsets containing these get included)
// We use word frequency as a proxy: only keep synsets where at least one word
// is "common enough" based on WordNet's tag count files.
func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: gensynonyms <wordnet-dict-dir> <output.json.gz>\n")
		os.Exit(1)
	}
	dictDir := os.Args[1]
	outFile := os.Args[2]

	// Load word frequency data from cntlist.rev (word -> tag count)
	freqs := loadFrequencies(dictDir + "/cntlist.rev")

	// Parse all data files
	var allGroups [][]string
	for _, pos := range []string{"noun", "verb", "adj", "adv"} {
		groups := parseDataFile(dictDir+"/data."+pos, freqs)
		allGroups = append(allGroups, groups...)
	}

	// Sort by size of group (larger groups first) for determinism
	sort.Slice(allGroups, func(i, j int) bool {
		if len(allGroups[i]) != len(allGroups[j]) {
			return len(allGroups[i]) > len(allGroups[j])
		}
		return allGroups[i][0] < allGroups[j][0]
	})

	fmt.Fprintf(os.Stderr, "Total synonym groups: %d\n", len(allGroups))

	// Write compressed JSON
	f, err := os.Create(outFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	gw, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	enc := json.NewEncoder(gw)
	if err := enc.Encode(allGroups); err != nil {
		panic(err)
	}

	fmt.Fprintf(os.Stderr, "Written to %s\n", outFile)
}

// loadFrequencies reads cntlist.rev to get word frequencies.
// Format: tag_cnt sense_key sense_number
func loadFrequencies(path string) map[string]int {
	freqs := make(map[string]int)
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot open %s: %v\n", path, err)
		return freqs
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		// Format: sense_key tag_cnt sense_number
		senseKey := parts[0]
		count, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		// sense_key format: word%ss_type:...
		word := strings.SplitN(senseKey, "%", 2)[0]
		word = strings.ReplaceAll(word, "_", " ")
		freqs[word] += count
	}
	return freqs
}

// parseDataFile parses a WordNet data file and returns synonym groups.
// Only returns groups with 2+ single-word members where at least one word
// has a frequency tag count >= minFreq.
func parseDataFile(path string, freqs map[string]int) [][]string {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	const minFreq = 3 // minimum tag count to consider a word "common"

	var groups [][]string
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		// WordNet data lines start with an 8-digit synset offset
		if len(line) < 20 || line[0] < '0' || line[0] > '9' {
			continue
		}
		// Skip header/license lines (they have spaces at start or short offset)
		if line[0] == ' ' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// Field 3 (0-indexed) is hex word count
		wordCountHex := fields[3]
		wordCount, err := strconv.ParseInt(wordCountHex, 16, 32)
		if err != nil || wordCount < 2 {
			continue // Skip single-word synsets
		}

		// Extract words: fields[4], fields[6], fields[8], etc.
		// Each word is followed by a lex_id
		var words []string
		hasCommon := false
		idx := 4
		for i := int64(0); i < wordCount && idx < len(fields); i++ {
			word := fields[idx]
			word = strings.ReplaceAll(word, "_", " ")
			word = strings.ToLower(word)

			// Skip multi-word phrases (contain spaces) — they bloat the list
			// and rarely match search queries
			if !strings.Contains(word, " ") {
				words = append(words, word)
				if freqs[word] >= minFreq {
					hasCommon = true
				}
			}
			idx += 2 // skip lex_id
		}

		// Only keep groups with 2+ single words and at least one common word
		if len(words) >= 2 && hasCommon {
			// Deduplicate words within the group
			seen := make(map[string]bool)
			var deduped []string
			for _, w := range words {
				if !seen[w] {
					seen[w] = true
					deduped = append(deduped, w)
				}
			}
			if len(deduped) >= 2 {
				sort.Strings(deduped)
				groups = append(groups, deduped)
			}
		}
	}

	return groups
}
