package search

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	bleve "github.com/blevesearch/bleve/v2"
)

//go:embed synonyms.json.gz
var synonymData []byte

const (
	synonymCollection = "wordnet"
	synonymSourceName = "english"
	synonymAnalyzer   = "en"
)

// synonymGroups caches the parsed synonym data.
var (
	synonymOnce   sync.Once
	synonymGroups [][]string
	synonymErr    error
)

// loadSynonyms parses the embedded synonym data (lazily, once).
func loadSynonyms() ([][]string, error) {
	synonymOnce.Do(func() {
		reader, err := gzip.NewReader(bytes.NewReader(synonymData))
		if err != nil {
			synonymErr = fmt.Errorf("decompress synonyms: %w", err)
			return
		}
		defer reader.Close()

		if err := json.NewDecoder(reader).Decode(&synonymGroups); err != nil {
			synonymErr = fmt.Errorf("parse synonyms: %w", err)
			return
		}
	})
	return synonymGroups, synonymErr
}

// indexSynonyms loads synonym groups and indexes them into a bleve index.
func indexSynonyms(idx bleve.Index) error {
	groups, err := loadSynonyms()
	if err != nil {
		return err
	}

	const batchSize = 500
	batch := idx.NewBatch()
	for i, group := range groups {
		def := &bleve.SynonymDefinition{
			Synonyms: group,
		}
		id := fmt.Sprintf("syn-%d", i)
		if err := batch.IndexSynonym(id, synonymCollection, def); err != nil {
			return fmt.Errorf("batch synonym group %d: %w", i, err)
		}
		if (i+1)%batchSize == 0 {
			if err := idx.Batch(batch); err != nil {
				return fmt.Errorf("flush synonym batch: %w", err)
			}
			batch = idx.NewBatch()
		}
	}
	// Flush remaining
	if batch.Size() > 0 {
		if err := idx.Batch(batch); err != nil {
			return fmt.Errorf("flush final synonym batch: %w", err)
		}
	}

	return nil
}
