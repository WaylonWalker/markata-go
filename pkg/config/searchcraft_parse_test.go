package config

import "testing"

func TestParseTOML_Searchcraft(t *testing.T) {
	data := []byte(`[markata-go]
title = "Example"

[markata-go.searchcraft]
enabled = true
endpoint = "http://localhost:18000"
ingest_key = "ingest"
read_key = "read"
index_prefix = "example"
index_per_site = true
`)

	cfg, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML failed: %v", err)
	}
	if cfg.Searchcraft.Enabled == nil || !*cfg.Searchcraft.Enabled {
		t.Fatalf("expected searchcraft enabled=true, got %#v", cfg.Searchcraft.Enabled)
	}
	if cfg.Searchcraft.Endpoint != "http://localhost:18000" {
		t.Fatalf("unexpected endpoint: %q", cfg.Searchcraft.Endpoint)
	}
	if cfg.Searchcraft.IndexPrefix != "example" {
		t.Fatalf("unexpected index_prefix: %q", cfg.Searchcraft.IndexPrefix)
	}
}
