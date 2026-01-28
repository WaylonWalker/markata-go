package buildcache

import (
	"reflect"
	"sort"
	"sync"
	"testing"
)

func TestNewDependencyGraph(t *testing.T) {
	g := NewDependencyGraph()

	if g == nil {
		t.Fatal("NewDependencyGraph returned nil")
	}
	if g.Dependencies == nil {
		t.Error("Dependencies map is nil")
	}
	if g.Dependents == nil {
		t.Error("Dependents map is nil")
	}
	if g.Size() != 0 {
		t.Errorf("Size() = %d, want 0", g.Size())
	}
}

func TestSetDependencies_Basic(t *testing.T) {
	g := NewDependencyGraph()

	// Set dependencies for post-a
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b", "post-c"})

	// Check forward dependencies
	deps := g.GetDependencies("pages/post-a.md")
	want := []string{"post-b", "post-c"}
	if !reflect.DeepEqual(deps, want) {
		t.Errorf("GetDependencies() = %v, want %v", deps, want)
	}

	// Check reverse dependencies
	if !g.HasDependents("post-b") {
		t.Error("post-b should have dependents")
	}
	if !g.HasDependents("post-c") {
		t.Error("post-c should have dependents")
	}

	dependents := g.GetDirectDependents("post-b")
	wantDeps := []string{"pages/post-a.md"}
	if !reflect.DeepEqual(dependents, wantDeps) {
		t.Errorf("GetDirectDependents(post-b) = %v, want %v", dependents, wantDeps)
	}
}

func TestSetDependencies_Deduplication(t *testing.T) {
	g := NewDependencyGraph()

	// Set dependencies with duplicates
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b", "post-c", "post-b", "post-c"})

	deps := g.GetDependencies("pages/post-a.md")
	want := []string{"post-b", "post-c"}
	if !reflect.DeepEqual(deps, want) {
		t.Errorf("GetDependencies() = %v, want %v (should be deduplicated)", deps, want)
	}
}

func TestSetDependencies_Replace(t *testing.T) {
	g := NewDependencyGraph()

	// Set initial dependencies
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b", "post-c"})

	// Replace with new dependencies
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-d"})

	// Check forward dependencies updated
	deps := g.GetDependencies("pages/post-a.md")
	want := []string{"post-d"}
	if !reflect.DeepEqual(deps, want) {
		t.Errorf("GetDependencies() = %v, want %v", deps, want)
	}

	// Check old reverse dependencies removed
	if g.HasDependents("post-b") {
		t.Error("post-b should not have dependents after replacement")
	}
	if g.HasDependents("post-c") {
		t.Error("post-c should not have dependents after replacement")
	}

	// Check new reverse dependency added
	if !g.HasDependents("post-d") {
		t.Error("post-d should have dependents")
	}
}

func TestSetDependencies_Empty(t *testing.T) {
	g := NewDependencyGraph()

	// Set then clear dependencies
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b"})
	g.SetDependencies("pages/post-a.md", "post-a", []string{})

	if g.HasDependencies("pages/post-a.md") {
		t.Error("post-a should not have dependencies after setting empty")
	}
	if g.HasDependents("post-b") {
		t.Error("post-b should not have dependents after clearing")
	}
	if g.Size() != 0 {
		t.Errorf("Size() = %d, want 0", g.Size())
	}
}

func TestGetAffectedPosts_Direct(t *testing.T) {
	g := NewDependencyGraph()

	// post-a -> post-b
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b"})

	// When post-b changes, post-a should be affected
	affected := g.GetAffectedPosts([]string{"post-b"})
	want := []string{"pages/post-a.md"}
	if !reflect.DeepEqual(affected, want) {
		t.Errorf("GetAffectedPosts([post-b]) = %v, want %v", affected, want)
	}
}

func TestGetAffectedPosts_Transitive(t *testing.T) {
	g := NewDependencyGraph()

	// post-a -> post-b -> post-c
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b"})
	g.SetDependencies("pages/post-b.md", "post-b", []string{"post-c"})

	// When post-c changes, both post-a and post-b should be affected
	affected := g.GetAffectedPosts([]string{"post-c"})
	sort.Strings(affected)
	want := []string{"pages/post-a.md", "pages/post-b.md"}
	if !reflect.DeepEqual(affected, want) {
		t.Errorf("GetAffectedPosts([post-c]) = %v, want %v", affected, want)
	}
}

func TestGetAffectedPosts_MultipleSources(t *testing.T) {
	g := NewDependencyGraph()

	// post-a -> post-c
	// post-b -> post-c
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-c"})
	g.SetDependencies("pages/post-b.md", "post-b", []string{"post-c"})

	// When post-c changes, both post-a and post-b should be affected
	affected := g.GetAffectedPosts([]string{"post-c"})
	sort.Strings(affected)
	want := []string{"pages/post-a.md", "pages/post-b.md"}
	if !reflect.DeepEqual(affected, want) {
		t.Errorf("GetAffectedPosts([post-c]) = %v, want %v", affected, want)
	}
}

func TestGetAffectedPosts_Diamond(t *testing.T) {
	g := NewDependencyGraph()

	// Diamond pattern:
	//     post-a
	//    /      \
	// post-b  post-c
	//    \      /
	//     post-d
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b", "post-c"})
	g.SetDependencies("pages/post-b.md", "post-b", []string{"post-d"})
	g.SetDependencies("pages/post-c.md", "post-c", []string{"post-d"})

	// When post-d changes, all others should be affected
	affected := g.GetAffectedPosts([]string{"post-d"})
	sort.Strings(affected)
	want := []string{"pages/post-a.md", "pages/post-b.md", "pages/post-c.md"}
	if !reflect.DeepEqual(affected, want) {
		t.Errorf("GetAffectedPosts([post-d]) = %v, want %v", affected, want)
	}
}

func TestGetAffectedPosts_Circular(t *testing.T) {
	g := NewDependencyGraph()

	// Circular: post-a -> post-b -> post-c -> post-a
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b"})
	g.SetDependencies("pages/post-b.md", "post-b", []string{"post-c"})
	g.SetDependencies("pages/post-c.md", "post-c", []string{"post-a"})

	// When post-a changes, should get all others without infinite loop
	affected := g.GetAffectedPosts([]string{"post-a"})
	sort.Strings(affected)
	want := []string{"pages/post-b.md", "pages/post-c.md"}
	if !reflect.DeepEqual(affected, want) {
		t.Errorf("GetAffectedPosts([post-a]) = %v, want %v", affected, want)
	}
}

func TestGetAffectedPosts_SelfReference(t *testing.T) {
	g := NewDependencyGraph()

	// post-a -> post-a (self-reference)
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-a"})

	// When post-a changes, it's already being rebuilt due to the change,
	// so it shouldn't be included in the affected posts list.
	// Self-references don't cause additional rebuilds.
	affected := g.GetAffectedPosts([]string{"post-a"})
	if len(affected) != 0 {
		t.Errorf("GetAffectedPosts([post-a]) = %v, want [] (self-reference shouldn't cause extra rebuild)", affected)
	}
}

func TestGetAffectedPosts_NoAffected(t *testing.T) {
	g := NewDependencyGraph()

	// post-a -> post-b (nothing depends on post-a)
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b"})

	// When post-a changes, nothing else is affected
	affected := g.GetAffectedPosts([]string{"post-a"})
	if len(affected) != 0 {
		t.Errorf("GetAffectedPosts([post-a]) = %v, want []", affected)
	}
}

func TestGetAffectedPosts_Empty(t *testing.T) {
	g := NewDependencyGraph()

	affected := g.GetAffectedPosts([]string{})
	if affected != nil {
		t.Errorf("GetAffectedPosts([]) = %v, want nil", affected)
	}

	affected = g.GetAffectedPosts(nil)
	if affected != nil {
		t.Errorf("GetAffectedPosts(nil) = %v, want nil", affected)
	}
}

func TestGetAffectedPosts_MultipleChanged(t *testing.T) {
	g := NewDependencyGraph()

	// post-a -> post-c
	// post-b -> post-d
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-c"})
	g.SetDependencies("pages/post-b.md", "post-b", []string{"post-d"})

	// When both post-c and post-d change
	affected := g.GetAffectedPosts([]string{"post-c", "post-d"})
	sort.Strings(affected)
	want := []string{"pages/post-a.md", "pages/post-b.md"}
	if !reflect.DeepEqual(affected, want) {
		t.Errorf("GetAffectedPosts([post-c, post-d]) = %v, want %v", affected, want)
	}
}

func TestRebuildReverse(t *testing.T) {
	g := NewDependencyGraph()

	// Set up dependencies
	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b", "post-c"})
	g.SetDependencies("pages/post-b.md", "post-b", []string{"post-c"})

	// Clear and rebuild reverse index
	g.mu.Lock()
	g.Dependents = make(map[string][]string)
	g.mu.Unlock()

	g.RebuildReverse()

	// Check reverse index is correct
	if !g.HasDependents("post-b") {
		t.Error("post-b should have dependents after rebuild")
	}
	if !g.HasDependents("post-c") {
		t.Error("post-c should have dependents after rebuild")
	}

	deps := g.GetDirectDependents("post-c")
	sort.Strings(deps)
	want := []string{"pages/post-a.md", "pages/post-b.md"}
	if !reflect.DeepEqual(deps, want) {
		t.Errorf("GetDirectDependents(post-c) = %v, want %v", deps, want)
	}
}

func TestRemoveSource(t *testing.T) {
	g := NewDependencyGraph()

	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b", "post-c"})
	g.SetDependencies("pages/post-d.md", "post-d", []string{"post-c"})

	g.RemoveSource("pages/post-a.md")

	// post-a should have no dependencies
	if g.HasDependencies("pages/post-a.md") {
		t.Error("post-a should have no dependencies after removal")
	}

	// post-b should have no dependents
	if g.HasDependents("post-b") {
		t.Error("post-b should have no dependents after removing post-a")
	}

	// post-c should still have post-d as dependent
	if !g.HasDependents("post-c") {
		t.Error("post-c should still have dependents (post-d)")
	}
}

func TestClear(t *testing.T) {
	g := NewDependencyGraph()

	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b", "post-c"})
	g.SetDependencies("pages/post-b.md", "post-b", []string{"post-c"})

	g.Clear()

	if g.Size() != 0 {
		t.Errorf("Size() = %d after Clear(), want 0", g.Size())
	}
	if g.HasDependencies("pages/post-a.md") {
		t.Error("post-a should have no dependencies after Clear()")
	}
	if g.HasDependents("post-c") {
		t.Error("post-c should have no dependents after Clear()")
	}
}

func TestConcurrency(t *testing.T) {
	g := NewDependencyGraph()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			source := "pages/post-" + string(rune('a'+id%26)) + ".md"
			slug := "post-" + string(rune('a'+id%26))
			targets := []string{"target-1", "target-2"}
			g.SetDependencies(source, slug, targets)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.GetAffectedPosts([]string{"target-1"})
			g.GetDirectDependents("target-2")
			g.HasDependencies("pages/post-a.md")
		}()
	}

	wg.Wait()

	// Should complete without deadlock or panic
	if g.Size() == 0 {
		t.Error("Expected some dependencies after concurrent writes")
	}
}

func TestGetDependencies_NotFound(t *testing.T) {
	g := NewDependencyGraph()

	deps := g.GetDependencies("nonexistent")
	if deps != nil {
		t.Errorf("GetDependencies(nonexistent) = %v, want nil", deps)
	}
}

func TestGetDirectDependents_NotFound(t *testing.T) {
	g := NewDependencyGraph()

	deps := g.GetDirectDependents("nonexistent")
	if deps != nil {
		t.Errorf("GetDirectDependents(nonexistent) = %v, want nil", deps)
	}
}

func TestSize(t *testing.T) {
	g := NewDependencyGraph()

	if g.Size() != 0 {
		t.Errorf("Size() = %d, want 0", g.Size())
	}

	g.SetDependencies("pages/post-a.md", "post-a", []string{"post-b"})
	if g.Size() != 1 {
		t.Errorf("Size() = %d, want 1", g.Size())
	}

	g.SetDependencies("pages/post-b.md", "post-b", []string{"post-c"})
	if g.Size() != 2 {
		t.Errorf("Size() = %d, want 2", g.Size())
	}

	g.RemoveSource("pages/post-a.md")
	if g.Size() != 1 {
		t.Errorf("Size() = %d, want 1 after removal", g.Size())
	}
}

// Benchmark tests
func BenchmarkSetDependencies(b *testing.B) {
	g := NewDependencyGraph()
	targets := []string{"post-1", "post-2", "post-3", "post-4", "post-5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.SetDependencies("pages/post-a.md", "post-a", targets)
	}
}

func BenchmarkGetAffectedPosts_Linear(b *testing.B) {
	g := NewDependencyGraph()

	// Create linear chain: post-0 -> post-1 -> ... -> post-99
	for i := 0; i < 100; i++ {
		source := "pages/post-" + string(rune('0'+i/10)) + string(rune('0'+i%10)) + ".md"
		slug := "post-" + string(rune('0'+i/10)) + string(rune('0'+i%10))
		target := "post-" + string(rune('0'+(i+1)/10)) + string(rune('0'+(i+1)%10))
		g.SetDependencies(source, slug, []string{target})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.GetAffectedPosts([]string{"post-99"})
	}
}

func BenchmarkGetAffectedPosts_Wide(b *testing.B) {
	g := NewDependencyGraph()

	// Create wide graph: 100 posts all depend on post-target
	for i := 0; i < 100; i++ {
		source := "pages/post-" + string(rune('0'+i/10)) + string(rune('0'+i%10)) + ".md"
		slug := "post-" + string(rune('0'+i/10)) + string(rune('0'+i%10))
		g.SetDependencies(source, slug, []string{"post-target"})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.GetAffectedPosts([]string{"post-target"})
	}
}
