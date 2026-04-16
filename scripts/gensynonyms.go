// Command gensynonyms parses WordNet data files and generates a curated
// synonym JSON file for embedding in the markata-go binary.
//
// It extracts synonym groups from two sources:
//  1. Within-synset synonyms (words in the same synset are synonyms)
//  2. Cross-POS derivational links ("+" and "\" pointers) that connect
//     related words across parts of speech (e.g., lunar adj → moon noun)
//
// Groups are merged via union-find when they share words, producing
// unified synonym clusters.
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

// synset holds parsed data for a single WordNet synset.
type synset struct {
	offset string
	pos    string
	words  []string // single-word, lowercased members
}

// derivLink represents a derivational or pertainym pointer between synsets.
type derivLink struct {
	fromOffset string
	fromPOS    string
	toOffset   string
	toPOS      string
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: gensynonyms <wordnet-dict-dir> <output.json.gz>\n")
		os.Exit(1)
	}
	dictDir := os.Args[1]
	outFile := os.Args[2]

	freqs := loadFrequencies(dictDir + "/cntlist.rev")

	// Phase 1: Parse all synsets and derivational links
	allSynsets := make(map[string]*synset) // key: "offset:pos"
	var allLinks []derivLink

	for _, pos := range []string{"noun", "verb", "adj", "adv"} {
		synsets, links := parseDataFile(dictDir+"/data."+pos, pos)
		for _, s := range synsets {
			key := s.offset + ":" + s.pos
			allSynsets[key] = s
		}
		allLinks = append(allLinks, links...)
	}
	fmt.Fprintf(os.Stderr, "Parsed %d synsets, %d derivational links\n", len(allSynsets), len(allLinks))

	const minFreq = 3

	// Phase 2: Build synonym groups from synsets with 2+ words (same as before)
	groupSet := make(map[string]bool) // deduplicate groups by sorted key
	var allGroups [][]string

	for _, s := range allSynsets {
		if len(s.words) < 2 {
			continue
		}
		hasCommon := false
		for _, w := range s.words {
			if freqs[w] >= minFreq {
				hasCommon = true
				break
			}
		}
		if hasCommon {
			deduped := dedup(s.words)
			if len(deduped) >= 2 {
				sort.Strings(deduped)
				key := strings.Join(deduped, "\x00")
				if !groupSet[key] {
					groupSet[key] = true
					allGroups = append(allGroups, deduped)
				}
			}
		}
	}
	fmt.Fprintf(os.Stderr, "Within-synset groups: %d\n", len(allGroups))

	// Phase 3: Create cross-POS pairs from derivational links.
	// Each link creates a small group of words from both synsets,
	// but we DON'T merge groups globally — this prevents chain explosions.
	derivGroups := 0
	for _, link := range allLinks {
		fromKey := link.fromOffset + ":" + link.fromPOS
		toKey := link.toOffset + ":" + link.toPOS
		fromSyn := allSynsets[fromKey]
		toSyn := allSynsets[toKey]
		if fromSyn == nil || toSyn == nil {
			continue
		}

		// Only create groups from single-word synsets crossing POS boundaries.
		// Multi-word synsets are already captured in Phase 2.
		// This keeps cross-POS groups small and focused.
		merged := make(map[string]bool)
		for _, w := range fromSyn.words {
			merged[w] = true
		}
		for _, w := range toSyn.words {
			merged[w] = true
		}
		if len(merged) < 2 {
			continue
		}
		hasCommon := false
		var words []string
		for w := range merged {
			words = append(words, w)
			if freqs[w] >= minFreq {
				hasCommon = true
			}
		}
		if !hasCommon {
			continue
		}
		sort.Strings(words)
		key := strings.Join(words, "\x00")
		if !groupSet[key] {
			groupSet[key] = true
			allGroups = append(allGroups, words)
			derivGroups++
		}
	}
	fmt.Fprintf(os.Stderr, "Cross-POS derivational groups: %d\n", derivGroups)

	// Sort for determinism
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
		senseKey := parts[0]
		count, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		word := strings.SplitN(senseKey, "%", 2)[0]
		word = strings.ReplaceAll(word, "_", " ")
		freqs[word] += count
	}
	return freqs
}

// parseDataFile parses a WordNet data file, returning synsets and derivational links.
func parseDataFile(path, posName string) ([]*synset, []derivLink) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	posCode := map[string]string{"noun": "n", "verb": "v", "adj": "a", "adv": "r"}[posName]

	var synsets []*synset
	var links []derivLink

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 20 || line[0] < '0' || line[0] > '9' || line[0] == ' ' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		offset := fields[0]
		wordCountHex := fields[3]
		wordCount, err := strconv.ParseInt(wordCountHex, 16, 32)
		if err != nil {
			continue
		}

		// Extract words
		var words []string
		idx := 4
		for i := int64(0); i < wordCount && idx < len(fields); i++ {
			word := fields[idx]
			word = strings.ReplaceAll(word, "_", " ")
			word = strings.ToLower(word)
			// Strip adjective syntax markers like (a), (p), (ip)
			if paren := strings.Index(word, "("); paren > 0 {
				word = word[:paren]
			}
			if !strings.Contains(word, " ") {
				words = append(words, word)
			}
			idx += 2
		}

		syn := &synset{offset: offset, pos: posCode, words: words}
		synsets = append(synsets, syn)

		// Parse pointers to find derivational links ("+" and "\")
		if idx >= len(fields) {
			continue
		}
		ptrCountStr := fields[idx]
		ptrCount, err := strconv.Atoi(ptrCountStr)
		if err != nil {
			continue
		}
		idx++

		for i := 0; i < ptrCount && idx+3 < len(fields); i++ {
			ptrType := fields[idx]
			targetOffset := fields[idx+1]
			targetPOS := fields[idx+2]
			// fields[idx+3] is source/target word indices
			idx += 4

			if ptrType == "+" || ptrType == `\` {
				links = append(links, derivLink{
					fromOffset: offset,
					fromPOS:    posCode,
					toOffset:   targetOffset,
					toPOS:      targetPOS,
				})
			}
		}
	}

	return synsets, links
}

// dedup removes duplicate strings, preserving order.
func dedup(words []string) []string {
	seen := make(map[string]bool, len(words))
	var result []string
	for _, w := range words {
		if !seen[w] {
			seen[w] = true
			result = append(result, w)
		}
	}
	return result
}
