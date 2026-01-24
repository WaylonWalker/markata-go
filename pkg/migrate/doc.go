// Package migrate provides tools for migrating from Python markata to markata-go.
//
// # Overview
//
// The migrate package helps users transition from Python markata by:
//   - Converting configuration files from Python markata format to markata-go format
//   - Migrating filter expressions to markata-go syntax
//   - Checking template compatibility with pongo2
//   - Generating detailed migration reports
//
// # Configuration Migration
//
// Configuration migration handles:
//   - Namespace changes ([markata] -> [markata-go])
//   - Key renames (glob_patterns -> patterns, etc.)
//   - Nav map to array conversion
//   - Feed filter expression migration
//
// # Filter Migration
//
// Filter expressions are migrated to handle:
//   - Boolean literal changes (published == 'True' -> published == True)
//   - `in` operator expansion (x in ['a', 'b'] -> x == 'a' or x == 'b')
//   - Operator spacing fixes (date<=today -> date <= today)
//
// # Usage
//
// Basic migration:
//
//	result, err := migrate.MigrateConfig("markata.toml", "markata-go.toml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Report())
//
// Filter migration:
//
//	migrated, changes := migrate.MigrateFilter("published == 'True'")
//	// migrated = "published == True"
//	// changes = ["Boolean literal: 'True' -> True"]
package migrate
