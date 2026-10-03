package builddag

import (
	"fmt"
	"testing"
)

// BenchmarkCompile measures compilation only, not declaration construction.
// Wide graphs have a root, a broad ready frontier, and a final fan-in barrier.
func BenchmarkCompile(b *testing.B) {
	for _, shape := range []string{"chain", "wide"} {
		for _, size := range []int{1000, 4000, 16000} {
			b.Run(fmt.Sprintf("%s/%d", shape, size), func(b *testing.B) {
				builder := NewBuilder()
				artifacts := make([]ArtifactID, size)
				for i := range artifacts {
					artifacts[i] = ArtifactID{Kind: "task", Key: fmt.Sprintf("%06d", i)}
				}
				for i := 0; i < size; i++ {
					task := TaskSpec{
						ID:       TaskID(artifacts[i].Key),
						Provides: []ArtifactID{artifacts[i]},
					}
					switch {
					case shape == "chain" && i > 0:
						task.Requires = []ArtifactID{artifacts[i-1]}
					case shape == "wide" && i == size-1:
						task.Requires = artifacts[1 : size-1]
					case shape == "wide" && i > 0:
						task.Requires = []ArtifactID{artifacts[0]}
					}
					builder.AddTask(task)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					graph, err := builder.Compile()
					if err != nil {
						b.Fatal(err)
					}
					if len(graph.order) != size {
						b.Fatalf("compiled %d tasks, want %d", len(graph.order), size)
					}
				}
			})
		}
	}
}
