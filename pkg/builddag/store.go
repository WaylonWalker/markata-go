package builddag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// MemoryStore is an explicit in-memory artifact store.
type MemoryStore struct {
	mu     sync.RWMutex
	values map[ArtifactID]Artifact
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{values: make(map[ArtifactID]Artifact)} }
func (s *MemoryStore) Get(id ArtifactID) (Artifact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.values[id]
	if !ok {
		return Artifact{}, false
	}
	a.Data = append([]byte(nil), a.Data...)
	return a, true
}
func (s *MemoryStore) Put(a Artifact) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a.Data = append([]byte(nil), a.Data...)
	s.values[a.ID] = a
}

// LoadExecutionState reads a previously saved scheduler state.
func LoadExecutionState(path string) (*ExecutionState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state ExecutionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode execution state: %w", err)
	}
	if state.Tasks == nil {
		state.Tasks = make(map[TaskID]TaskState)
	}
	return &state, nil
}

// SaveExecutionState writes scheduler state with an atomic same-directory
// replacement so an interrupted build cannot leave a partial state file.
func SaveExecutionState(path string, state *ExecutionState) error {
	if path == "" {
		return fmt.Errorf("execution state path is empty")
	}
	if state == nil {
		return fmt.Errorf("execution state is nil")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode execution state: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create execution state directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".builddag-state-*")
	if err != nil {
		return fmt.Errorf("create execution state temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set execution state permissions: %w", err)
	}
	if _, err = tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write execution state: %w", err)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write execution state: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace execution state: %w", err)
	}
	return nil
}
