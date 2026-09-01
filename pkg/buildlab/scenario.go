package buildlab

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type OperationType string

const (
	OpBuild        OperationType = "build"
	OpWriteFile    OperationType = "write-file"
	OpReplaceExact OperationType = "replace-exact"
	OpDelete       OperationType = "delete-file"
	OpRename       OperationType = "rename-file"
	OpCopy         OperationType = "copy-file"
	OpSetConfig    OperationType = "set-config"
	OpTouch        OperationType = "touch-file"
	OpCleanCache   OperationType = "clean-cache"
	// OpClearCache is kept as a source-compatible name for callers that used
	// the original Go constant. The JSON operation name is clean-cache.
	OpClearCache  OperationType = OpCleanCache
	OpClearOutput OperationType = "clear-output"

	opLegacyClearCache OperationType = "clear-cache"
)

type Operation struct {
	Type    OperationType `json:"type"`
	Path    string        `json:"path,omitempty"`
	Dest    string        `json:"dest,omitempty"`
	Content string        `json:"content,omitempty"`
	Old     string        `json:"old,omitempty"`
	New     string        `json:"new,omitempty"`
	Key     string        `json:"key,omitempty"`
	Value   string        `json:"value,omitempty"`
}
type Scenario struct {
	ID         string      `json:"id"`
	Version    string      `json:"version"`
	Seed       int64       `json:"seed"`
	Operations []Operation `json:"operations"`
}
type ScenarioError struct {
	Operation int
	Err       error
}

func (e *ScenarioError) Error() string {
	return fmt.Sprintf("scenario operation %d: %v", e.Operation, e.Err)
}
func (e *ScenarioError) Unwrap() error { return e.Err }

var ErrPrecondition = errors.New("replace precondition failed")

// Validate checks the required versioned scenario contract. JSON decoding
// additionally checks that required fields are present, because a zero seed
// is valid and cannot be distinguished from an omitted seed after decoding.
func (s Scenario) Validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return errors.New("scenario id is required")
	}
	if s.Version == "" {
		return errors.New("scenario version is required")
	}
	if s.Version != "1" {
		return fmt.Errorf("unsupported scenario version %q", s.Version)
	}
	if s.Operations == nil {
		return errors.New("scenario operations must be an array")
	}
	for i := range s.Operations {
		operation := &s.Operations[i]
		if !knownOperation(operation.Type) {
			return fmt.Errorf("scenario operation %d: unknown operation %q", i, operation.Type)
		}
		if err := validateOperation(*operation); err != nil {
			return fmt.Errorf("scenario operation %d (%s): %w", i, operation.Type, err)
		}
	}
	return nil
}

// validateOperation checks the shape of an operation before a fixture is
// copied or mutated. It intentionally does not check whether a path exists:
// that is a property of the fixture state at the point the operation runs.
//
//nolint:gocyclo // Each operation type has a small, explicit payload contract.
func validateOperation(operation Operation) error {
	switch operation.Type {
	case OpBuild, OpCleanCache, opLegacyClearCache, OpClearOutput:
		if operation.Path != "" || operation.Dest != "" || operation.Content != "" || operation.Old != "" || operation.New != "" || operation.Key != "" || operation.Value != "" {
			return errors.New("operation does not accept payload fields")
		}
	case OpWriteFile:
		if operation.Path == "" {
			return errors.New("path is required")
		}
	case OpReplaceExact:
		if operation.Path == "" {
			return errors.New("path is required")
		}
		if operation.Old == "" {
			return errors.New("old text is required")
		}
	case OpDelete, OpTouch:
		if operation.Path == "" {
			return errors.New("path is required")
		}
	case OpRename, OpCopy:
		if operation.Path == "" || operation.Dest == "" {
			return errors.New("path and dest are required")
		}
	case OpSetConfig:
		if operation.Path == "" {
			return errors.New("path is required")
		}
		if operation.Key == "" || strings.ContainsAny(operation.Key, "\r\n= \t") {
			return errors.New("key is required and must be a single config key")
		}
	}
	return nil
}

func knownOperation(operation OperationType) bool {
	switch operation {
	case OpBuild, OpWriteFile, OpReplaceExact, OpDelete, OpRename, OpCopy,
		OpSetConfig, OpTouch, OpCleanCache, OpClearOutput, opLegacyClearCache:
		return true
	default:
		return false
	}
}

func safePath(root, name string) (string, error) {
	if name == "" || filepath.IsAbs(name) {
		return "", fmt.Errorf("unsafe path %q", name)
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe path %q", name)
	}
	result := filepath.Join(root, clean)
	// Do not allow an existing symlink in a parent directory to redirect an
	// operation outside the workspace. The final path is checked by operations
	// which write through it (rename/delete may intentionally address a link).
	part := root
	components := strings.Split(clean, string(filepath.Separator))
	for _, component := range components[:len(components)-1] {
		part = filepath.Join(part, component)
		info, err := os.Lstat(part)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("path %q enters a symlink", name)
		}
	}
	return result, nil
}
func removeTree(root, name string) error {
	p, e := safePath(root, name)
	if e != nil {
		return e
	}
	return os.RemoveAll(p)
}

// CanonicalJSON returns the stable representation used when persisting a
// scenario or reproducing a generated failure.
func (s Scenario) CanonicalJSON() ([]byte, error) {
	if s.Version == "" {
		s.Version = "1"
	}
	if s.Operations == nil {
		s.Operations = []Operation{}
	}
	return json.Marshal(s)
}

// Digest returns the SHA-256 digest of the canonical scenario JSON.
func (s Scenario) Digest() (string, error) {
	b, err := s.CanonicalJSON()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func rejectSymlink(path string) error {
	info, err := os.Lstat(path)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("path %q is a symlink", path)
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ApplyOperation applies one declarative operation. Build is intentionally a
// marker/no-op; the caller owns process execution.
func ApplyOperation(root string, op Operation) error {
	if op.Type == OpCleanCache || op.Type == opLegacyClearCache {
		for _, name := range []string{".markata", ".markata.cache", ".markata-cache", "cache"} {
			if err := removeTree(root, name); err != nil {
				return err
			}
		}
		return nil
	}
	if op.Type == OpClearOutput {
		return removeTree(root, "output")
	}
	p, err := safePath(root, op.Path)
	if err != nil && op.Type != OpBuild {
		return err
	}
	switch op.Type {
	case OpBuild:
		return nil
	case OpWriteFile:
		return writeFileOperation(p, op.Content)
	case OpReplaceExact:
		return replaceExactOperation(p, op)
	case OpDelete:
		return os.Remove(p)
	case OpRename:
		return renameOperation(root, p, op.Dest)
	case OpCopy:
		return copyOperation(root, p, op.Dest)
	case OpSetConfig:
		return setConfigOperation(p, op)
	case OpTouch:
		return touchOperation(p)
	default:
		return fmt.Errorf("unknown operation %q", op.Type)
	}
}

func writeFileOperation(path, content string) error {
	if err := rejectSymlink(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func replaceExactOperation(path string, op Operation) error {
	if err := rejectSymlink(path); err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if occurrenceCount(string(b), op.Old) != 1 {
		return fmt.Errorf("%w: %s", ErrPrecondition, op.Path)
	}
	return os.WriteFile(path, []byte(strings.Replace(string(b), op.Old, op.New, 1)), 0o600)
}

func renameOperation(root, source, destination string) error {
	target, err := safePath(root, destination)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.Rename(source, target)
}

func copyOperation(root, source, destination string) error {
	if err := rejectSymlink(source); err != nil {
		return err
	}
	target, err := safePath(root, destination)
	if err != nil {
		return err
	}
	if err := rejectSymlink(target); err != nil {
		return err
	}
	b, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, b, 0o600)
}

func setConfigOperation(path string, op Operation) error {
	if op.Key == "" || strings.ContainsAny(op.Key, "\r\n= \t") {
		return fmt.Errorf("unsafe config key %q", op.Key)
	}
	if err := rejectSymlink(path); err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), op.Key+" =") {
			lines[i] = op.Key + " = " + quoteConfig(op.Value)
			replaced = true
		}
	}
	if !replaced {
		if len(lines) == 1 && lines[0] == "" {
			lines = nil
		}
		lines = append(lines, op.Key+" = "+quoteConfig(op.Value))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}

func touchOperation(path string) error {
	if err := rejectSymlink(path); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	now := time.Now()
	return os.Chtimes(path, now, now)
}
func quoteConfig(v string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), `"`, `\"`) + `"`
}

func occurrenceCount(s, needle string) int {
	if needle == "" {
		return 0
	}
	n := 0
	for i := 0; ; {
		j := strings.Index(s[i:], needle)
		if j < 0 {
			return n
		}
		n++
		i += j + 1 // count overlapping matches too
		if i >= len(s) {
			return n
		}
	}
}
func (s Scenario) Apply(root string) error {
	for i := range s.Operations {
		op := s.Operations[i]
		if err := ApplyOperation(root, op); err != nil {
			return &ScenarioError{Operation: i, Err: err}
		}
	}
	return nil
}
