// Package buildcache provides incremental build caching for markata-go.
package buildcache

import (
	"sort"
	"sync"
)

// DependencyGraph tracks relationships between posts for incremental builds.
// It maintains both forward dependencies (what a post links to) and reverse
// dependencies (what posts link to a given post) for efficient invalidation.
//
// Example:
//
//	post-a.md contains [[post-b]] and [[post-c]]
//	post-b.md contains [[post-c]]
//
//	Dependencies (forward):
//	  "post-a" -> ["post-b", "post-c"]
//	  "post-b" -> ["post-c"]
//
//	Dependents (reverse, computed):
//	  "post-b" -> ["post-a"]
//	  "post-c" -> ["post-a", "post-b"]
//
// When post-c changes, GetAffectedPosts returns ["post-a", "post-b"] (transitive).
type DependencyGraph struct {
	mu sync.RWMutex

	// Dependencies maps source path -> target slugs (what this post links TO)
	// Key is the source file path, values are slugs of linked posts
	Dependencies map[string][]string `json:"dependencies,omitempty"`

	// PathToSlug maps source path -> its slug (for reverse lookups during traversal)
	PathToSlug map[string]string `json:"path_to_slug,omitempty"`

	// Dependents maps target slug -> source paths (who links to this post)
	// This is computed from Dependencies and not persisted
	Dependents map[string][]string `json:"-"`
}

// NewDependencyGraph creates a new empty dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Dependencies: make(map[string][]string),
		PathToSlug:   make(map[string]string),
		Dependents:   make(map[string][]string),
	}
}

// SetDependencies records what targets a source post links to.
// This replaces any existing dependencies for the source.
// The sourceSlug is the slug of the source post (for reverse lookups).
// The targets are slugs of linked posts.
func (g *DependencyGraph) SetDependencies(sourcePath, sourceSlug string, targets []string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Remove old dependencies from reverse index
	if oldTargets, ok := g.Dependencies[sourcePath]; ok {
		for _, target := range oldTargets {
			g.removeDependent(target, sourcePath)
		}
	}

	// Store path-to-slug mapping
	if sourceSlug != "" {
		g.PathToSlug[sourcePath] = sourceSlug
	}

	// Store new dependencies
	if len(targets) == 0 {
		delete(g.Dependencies, sourcePath)
	} else {
		// Deduplicate and sort for deterministic output
		seen := make(map[string]bool, len(targets))
		unique := make([]string, 0, len(targets))
		for _, t := range targets {
			if !seen[t] {
				seen[t] = true
				unique = append(unique, t)
			}
		}
		sort.Strings(unique)
		g.Dependencies[sourcePath] = unique

		// Update reverse index
		for _, target := range unique {
			g.addDependent(target, sourcePath)
		}
	}
}

// GetDependencies returns the targets that a source post links to.
func (g *DependencyGraph) GetDependencies(sourcePath string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if deps, ok := g.Dependencies[sourcePath]; ok {
		result := make([]string, len(deps))
		copy(result, deps)
		return result
	}
	return nil
}

// SlugForPath returns the slug associated with a source path, if known.
func (g *DependencyGraph) SlugForPath(sourcePath string) string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.PathToSlug[sourcePath]
}

// GetDirectDependents returns posts that directly link to the given target.
// The target can be a slug or path.
func (g *DependencyGraph) GetDirectDependents(target string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if deps, ok := g.Dependents[target]; ok {
		result := make([]string, len(deps))
		copy(result, deps)
		return result
	}
	return nil
}

// GetAffectedPosts returns all posts that need to be rebuilt when the given
// posts change. This performs a transitive closure using BFS to find all
// posts that directly or indirectly depend on the changed posts.
//
// The input is a list of changed post slugs.
// The output is a list of source paths that need rebuilding.
// The changed posts themselves are NOT included in the result.
//
// Example: If A->B->C and C changes, returns [A, B] (both depend on C).
func (g *DependencyGraph) GetAffectedPosts(changed []string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(changed) == 0 {
		return nil
	}

	// Build set of changed slugs for filtering
	changedSlugs := make(map[string]bool, len(changed))
	for _, slug := range changed {
		changedSlugs[slug] = true
	}

	// Track visited to avoid cycles and duplicates
	visited := make(map[string]bool)
	affected := make(map[string]bool)

	// BFS queue - start with direct dependents of changed posts
	queue := make([]string, 0, len(changed)*4)

	// Seed queue with direct dependents of all changed posts
	for _, changedPost := range changed {
		visited[changedPost] = true
		if deps, ok := g.Dependents[changedPost]; ok {
			for _, dep := range deps {
				if !visited[dep] {
					queue = append(queue, dep)
					visited[dep] = true
				}
			}
		}
	}

	// BFS to find transitive dependents
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		affected[current] = true

		// Current is a source path. To find what depends on it transitively,
		// we need to look up its slug and find dependents of that slug.
		if slug, ok := g.PathToSlug[current]; ok {
			if deps, ok := g.Dependents[slug]; ok {
				for _, dep := range deps {
					if !visited[dep] {
						visited[dep] = true
						queue = append(queue, dep)
					}
				}
			}
		}
	}

	// Convert to sorted slice for deterministic output
	// Filter out paths whose slug is in the changed set
	result := make([]string, 0, len(affected))
	for path := range affected {
		// Don't include the changed post's path in the result
		if slug, ok := g.PathToSlug[path]; ok {
			if changedSlugs[slug] {
				continue
			}
		}
		result = append(result, path)
	}
	sort.Strings(result)

	return result
}

// RebuildReverse reconstructs the Dependents map from Dependencies.
// This should be called after loading the graph from disk.
func (g *DependencyGraph) RebuildReverse() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Dependents = make(map[string][]string)

	for source, targets := range g.Dependencies {
		for _, target := range targets {
			g.addDependent(target, source)
		}
	}
}

// Clear removes all dependencies from the graph.
func (g *DependencyGraph) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Dependencies = make(map[string][]string)
	g.PathToSlug = make(map[string]string)
	g.Dependents = make(map[string][]string)
}

// RemoveSource removes all dependencies for a source post.
// Use this when a post is deleted.
func (g *DependencyGraph) RemoveSource(sourcePath string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if targets, ok := g.Dependencies[sourcePath]; ok {
		for _, target := range targets {
			g.removeDependent(target, sourcePath)
		}
		delete(g.Dependencies, sourcePath)
	}
	delete(g.PathToSlug, sourcePath)
}

// Size returns the number of source posts with dependencies.
func (g *DependencyGraph) Size() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.Dependencies)
}

// addDependent adds a source to the dependents list for a target.
// Must be called with lock held.
func (g *DependencyGraph) addDependent(target, source string) {
	deps := g.Dependents[target]
	// Check if already present
	for _, d := range deps {
		if d == source {
			return
		}
	}
	g.Dependents[target] = append(deps, source)
}

// removeDependent removes a source from the dependents list for a target.
// Must be called with lock held.
func (g *DependencyGraph) removeDependent(target, source string) {
	deps := g.Dependents[target]
	for i, d := range deps {
		if d == source {
			// Remove by swapping with last and truncating
			deps[i] = deps[len(deps)-1]
			g.Dependents[target] = deps[:len(deps)-1]
			if len(g.Dependents[target]) == 0 {
				delete(g.Dependents, target)
			}
			return
		}
	}
}

// HasDependencies returns true if the source has any dependencies.
func (g *DependencyGraph) HasDependencies(sourcePath string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	deps, ok := g.Dependencies[sourcePath]
	return ok && len(deps) > 0
}

// HasDependents returns true if any posts depend on the target.
func (g *DependencyGraph) HasDependents(target string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	deps, ok := g.Dependents[target]
	return ok && len(deps) > 0
}
