package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// BakeValue is one setting written by BakeValues.
// Value must be a string, bool, int, int64, float64, or []string.
type BakeValue struct {
	Key   string
	Value any
}

// BakeSetting is one setting written by BakeSettings. Path is the key path
// below the markata-go wrapper, e.g. ["theme", "switcher", "enabled"].
type BakeSetting struct {
	Path  []string
	Value any
}

// ErrBakeUnsupportedLayout is returned when a config file expresses the target
// group in a shape that cannot be edited without rewriting user formatting
// (for example a TOML inline table or a YAML flow mapping).
var ErrBakeUnsupportedLayout = errors.New("config group layout cannot be edited safely")

// BakeRemove is a BakeSetting value that deletes the key from the file, so
// the setting falls back to its default (or to a lower-precedence file).
var BakeRemove any = bakeRemove{}

type bakeRemove struct{}

// IsBakeRemove reports whether value is BakeRemove.
func IsBakeRemove(value any) bool {
	_, ok := value.(bakeRemove)
	return ok
}

// withoutRemovals returns the values that set a key.
func withoutRemovals(values []BakeValue) []BakeValue {
	out := make([]BakeValue, 0, len(values))
	for _, v := range values {
		if !IsBakeRemove(v.Value) {
			out = append(out, v)
		}
	}
	return out
}

var bakeKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// FindGroupConfigFile returns the file in paths that should receive settings
// for markata-go.<group>. paths must be ordered from lowest to highest
// precedence (root config, then includes, then merge files). The last file
// that already defines the group wins, because a later file overrides earlier
// ones. An empty result means no file defines the group.
func FindGroupConfigFile(paths []string, group string) (string, error) {
	target := ""
	for _, path := range paths {
		raw, err := loadRawConfigFile(path)
		if err != nil {
			return "", err
		}
		if markata, ok := raw["markata-go"].(map[string]any); ok {
			if _, ok := markata[group]; ok {
				target = path
			}
		}
	}
	return target, nil
}

// SettingDefinedDepth reports how many leading segments of keyPath (below the
// markata-go wrapper) are defined in raw, a parsed config file. A result of
// len(keyPath) means the setting itself is defined.
func SettingDefinedDepth(raw map[string]any, keyPath []string) int {
	node, ok := raw["markata-go"].(map[string]any)
	if !ok {
		return 0
	}
	depth := 0
	for i, key := range keyPath {
		value, exists := node[key]
		if !exists {
			return depth
		}
		depth++
		if i == len(keyPath)-1 {
			return depth
		}
		next, isMap := value.(map[string]any)
		if !isMap {
			return depth
		}
		node = next
	}
	return depth
}

// LoadRawConfigFile parses a config file into a generic map without applying
// defaults, includes, or environment overrides.
func LoadRawConfigFile(path string) (map[string]any, error) {
	return loadRawConfigFile(path)
}

// BakeValues writes values into the markata-go.<group> section of the config
// file at path. See BakeSettings.
func BakeValues(path, group string, values []BakeValue) error {
	if !bakeKeyPattern.MatchString(group) {
		return fmt.Errorf("invalid config group %q", group)
	}
	settings := make([]BakeSetting, 0, len(values))
	for _, v := range values {
		settings = append(settings, BakeSetting{Path: []string{group, v.Key}, Value: v.Value})
	}
	return BakeSettings(path, settings)
}

// BakeSettings writes settings into the config file at path. When path does
// not exist it is created. Existing formatting, comments, and key order are
// preserved: matching keys are replaced in place (or deleted for BakeRemove),
// missing keys are added to their table, and absent tables are appended. The
// edited document is re-parsed and must differ from the original only by the
// baked keys, otherwise the file is left untouched.
func BakeSettings(path string, settings []BakeSetting) error {
	plan, err := PlanBake(path, settings)
	if err != nil {
		return err
	}
	return plan.Write()
}

// BakePlan is the verified result of editing one config file, ready to be
// written. Before is nil when the file does not exist yet.
type BakePlan struct {
	Path   string
	Exists bool
	Before []byte
	After  []byte
	mode   os.FileMode
}

// Changed reports whether writing the plan would modify the file.
func (p BakePlan) Changed() bool {
	if !p.Exists {
		return len(p.After) > 0
	}
	return !bytes.Equal(p.Before, p.After)
}

// Write replaces the file atomically with the planned content. A plan that
// changes nothing is a no-op.
func (p BakePlan) Write() error {
	if !p.Changed() {
		return nil
	}
	return WriteFileAtomic(p.Path, p.After, p.mode)
}

// PlanBake computes and verifies the edit BakeSettings would make to the
// config file at path without writing it.
func PlanBake(path string, settings []BakeSetting) (BakePlan, error) {
	plan := BakePlan{Path: path, mode: 0o644}
	if len(settings) == 0 {
		return plan, errors.New("no settings to bake")
	}
	type tableValues struct {
		table  []string
		values []BakeValue
	}
	var tables []*tableValues
	for _, s := range settings {
		if len(s.Path) == 0 {
			return plan, errors.New("empty config key")
		}
		for _, part := range s.Path {
			if !bakeKeyPattern.MatchString(part) {
				return plan, fmt.Errorf("invalid config key %q", strings.Join(s.Path, "."))
			}
		}
		if err := checkBakeValue(s.Value); err != nil {
			return plan, fmt.Errorf("%s: %w", strings.Join(s.Path, "."), err)
		}
		table, leaf := s.Path[:len(s.Path)-1], s.Path[len(s.Path)-1]
		var tv *tableValues
		for _, existing := range tables {
			if slices.Equal(existing.table, table) {
				tv = existing
				break
			}
		}
		if tv == nil {
			tv = &tableValues{table: table}
			tables = append(tables, tv)
		}
		tv.values = append(tv.values, BakeValue{Key: leaf, Value: s.Value})
	}

	format := detectConfigFormat(path)
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		plan.Exists = true
		plan.Before = data
		if info, statErr := os.Stat(path); statErr == nil {
			plan.mode = info.Mode().Perm()
		}
	case errors.Is(err, os.ErrNotExist):
		data = nil
	default:
		return plan, fmt.Errorf("read config file: %w", err)
	}

	before := map[string]any{}
	if len(bytes.TrimSpace(data)) > 0 {
		before, err = loadRawConfigData(data, format)
		if err != nil {
			return plan, fmt.Errorf("parse config file %s: %w", path, err)
		}
	}

	updated := data
	for _, tv := range tables {
		switch format {
		case FormatTOML:
			updated, err = bakeTOML(updated, tv.table, tv.values)
		case FormatYAML:
			updated, err = bakeYAML(updated, tv.table, tv.values)
		case FormatJSON:
			updated, err = bakeJSON(updated, tv.table, tv.values)
		default:
			err = fmt.Errorf("unsupported config format: %s", format)
		}
		if err != nil {
			return plan, fmt.Errorf("bake %s: %w", path, err)
		}
	}

	if err := verifyBake(before, updated, format, settings); err != nil {
		return plan, fmt.Errorf("bake %s: %w", path, err)
	}
	plan.After = updated
	return plan, nil
}

func checkBakeValue(value any) error {
	switch v := value.(type) {
	case bakeRemove, string, bool, int, int64, []string:
		return nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return errors.New("number must be finite")
		}
		return nil
	default:
		return fmt.Errorf("unsupported value type %T", value)
	}
}

// verifyBake re-parses the edited document and checks that applying settings
// to the original document yields exactly the edited document. Tables left
// empty (or null in YAML) count as absent.
func verifyBake(before map[string]any, updated []byte, format Format, settings []BakeSetting) error {
	after := map[string]any{}
	if len(bytes.TrimSpace(updated)) > 0 {
		var err error
		after, err = loadRawConfigData(updated, format)
		if err != nil {
			return fmt.Errorf("edited config does not parse: %w", err)
		}
	}
	expected := cloneMap(before)
	if expected == nil {
		expected = map[string]any{}
	}
	for _, s := range settings {
		parents := append([]string{"markata-go"}, s.Path[:len(s.Path)-1]...)
		leaf := s.Path[len(s.Path)-1]
		if IsBakeRemove(s.Value) {
			node := expected
			for _, key := range parents {
				child, ok := node[key].(map[string]any)
				if !ok {
					node = nil
					break
				}
				node = child
			}
			if node != nil {
				delete(node, leaf)
			}
			continue
		}
		node := expected
		for _, key := range parents {
			child, ok := node[key].(map[string]any)
			if !ok {
				child = map[string]any{}
			}
			node[key] = child
			node = child
		}
		node[leaf] = s.Value
	}
	if !reflect.DeepEqual(pruneEmptyTables(bakeComparable(expected)), pruneEmptyTables(bakeComparable(after))) {
		return errors.New("edited config did not match the expected settings; file left unchanged")
	}
	return nil
}

// pruneEmptyTables drops nil values and empty maps, recursively.
func pruneEmptyTables(value any) any {
	m, ok := value.(map[string]any)
	if !ok {
		return value
	}
	out := make(map[string]any, len(m))
	for key, item := range m {
		item = pruneEmptyTables(item)
		if item == nil {
			continue
		}
		if child, isMap := item.(map[string]any); isMap && len(child) == 0 {
			continue
		}
		out[key] = item
	}
	return out
}

// bakeComparable normalizes parsed values so that documents decoded by
// different parsers compare equal (all numbers become float64).
func bakeComparable(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = bakeComparable(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = bakeComparable(item)
		}
		return out
	case []string:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() { //nolint:exhaustive // Only numeric kinds are normalized.
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	case reflect.Map:
		return bakeComparable(normalizeValue(value))
	case reflect.Slice:
		if rv.Type().Elem().Kind() != reflect.Interface {
			out := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				out[i] = bakeComparable(rv.Index(i).Interface())
			}
			return out
		}
	}
	return value
}

// WriteFileAtomic replaces path with data via a temporary file and rename, so
// readers never see a partially written file.
func WriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".bake-*")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace config file: %w", err)
	}
	return nil
}

func bakeQuote(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04X`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func bakeScalar(value any) string {
	switch v := value.(type) {
	case string:
		return bakeQuote(v)
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		text := strconv.FormatFloat(v, 'f', -1, 64)
		if !strings.ContainsAny(text, ".eE") {
			text += ".0"
		}
		return text
	case []string:
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = bakeQuote(item)
		}
		return "[" + strings.Join(items, ", ") + "]"
	default:
		return bakeQuote(fmt.Sprint(v))
	}
}

func splitLinesKeepEnds(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	text := string(data)
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func detectNewline(data []byte) string {
	if bytes.Contains(data, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func lineEnding(line string) (body, ending string) {
	switch {
	case strings.HasSuffix(line, "\r\n"):
		return line[:len(line)-2], "\r\n"
	case strings.HasSuffix(line, "\n"):
		return line[:len(line)-1], "\n"
	default:
		return line, ""
	}
}

// ---------------------------------------------------------------------------
// TOML
// ---------------------------------------------------------------------------

var (
	tomlHeaderPattern = regexp.MustCompile(`^\s*\[\s*([^\[\]]+?)\s*\]\s*(#.*)?$`)
	tomlKeyPattern    = regexp.MustCompile(`^(\s*)((?:"[^"]*"|'[^']*'|[A-Za-z0-9_-]+)(?:\s*\.\s*(?:"[^"]*"|'[^']*'|[A-Za-z0-9_-]+))*)\s*=\s*(.*)$`)
)

type tomlLine struct {
	header    []string // table path when the line is a [table] header
	arrayHead bool     // [[array]] header
	keyIndent string
	key       []string // key path for key = value lines (depth 0 only)
	value     string   // raw value text after '='
}

func tomlKeyParts(raw string) []string {
	parts := strings.Split(raw, ".")
	for i, part := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(part), `"'`)
	}
	return parts
}

// scanTOML classifies top-level lines, skipping lines inside multi-line
// strings and multi-line arrays or inline tables.
func scanTOML(lines []string) []tomlLine {
	out := make([]tomlLine, len(lines))
	depth := 0
	multi := ""
	for i, line := range lines {
		body, _ := lineEnding(line)
		if multi != "" {
			if strings.Count(body, multi)%2 == 1 {
				multi = ""
			}
			continue
		}
		if depth > 0 {
			depth += tomlBracketDelta(body)
			if depth < 0 {
				depth = 0
			}
			continue
		}
		trimmed := strings.TrimSpace(body)
		if strings.HasPrefix(trimmed, "[[") {
			out[i].arrayHead = true
			continue
		}
		if m := tomlHeaderPattern.FindStringSubmatch(body); m != nil {
			out[i].header = tomlKeyParts(m[1])
			continue
		}
		m := tomlKeyPattern.FindStringSubmatch(body)
		if m == nil {
			continue
		}
		out[i].keyIndent = m[1]
		out[i].key = tomlKeyParts(m[2])
		out[i].value = m[3]
		value := strings.TrimSpace(m[3])
		for _, delim := range []string{`"""`, `'''`} {
			if strings.HasPrefix(value, delim) && strings.Count(value, delim)%2 == 1 {
				multi = delim
			}
		}
		if multi == "" {
			depth = tomlBracketDelta(value)
			if depth < 0 {
				depth = 0
			}
		}
	}
	return out
}

// tomlBracketDelta counts unbalanced [ and { outside strings and comments.
func tomlBracketDelta(text string) int {
	delta := 0
	var quote byte
	for i := 0; i < len(text); i++ {
		c := text[i]
		if quote != 0 {
			if c == '\\' && quote == '"' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
		case '#':
			return delta
		case '[', '{':
			delta++
		case ']', '}':
			delta--
		}
	}
	return delta
}

// tomlTrailingComment returns the comment after a simple scalar value, or
// ok=false when the value is not a single-line scalar.
func tomlTrailingComment(value string) (comment string, ok bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	rest := ""
	switch {
	case strings.HasPrefix(value, `"""`), strings.HasPrefix(value, `'''`),
		strings.HasPrefix(value, "["), strings.HasPrefix(value, "{"):
		return "", false
	case value[0] == '"':
		end := -1
		for i := 1; i < len(value); i++ {
			if value[i] == '\\' {
				i++
				continue
			}
			if value[i] == '"' {
				end = i
				break
			}
		}
		if end < 0 {
			return "", false
		}
		rest = value[end+1:]
	case value[0] == '\'':
		end := strings.IndexByte(value[1:], '\'')
		if end < 0 {
			return "", false
		}
		rest = value[end+2:]
	default:
		if idx := strings.IndexByte(value, '#'); idx >= 0 {
			rest = value[idx:]
		}
	}
	rest = strings.TrimSpace(rest)
	if rest != "" && !strings.HasPrefix(rest, "#") {
		return "", false
	}
	return rest, true
}

// tomlRegion is a run of lines owned by one [table] header. The root region
// (keys before the first header) has start -1 and a nil path.
type tomlRegion struct {
	path       []string
	start, end int
	array      bool
}

func tomlRegions(scanned []tomlLine) []tomlRegion {
	regions := []tomlRegion{{start: -1, end: len(scanned)}}
	for i, l := range scanned {
		if l.header == nil && !l.arrayHead {
			continue
		}
		regions[len(regions)-1].end = i
		regions = append(regions, tomlRegion{path: l.header, start: i, end: len(scanned), array: l.arrayHead})
	}
	return regions
}

func findTOMLRegion(regions []tomlRegion, path []string) (tomlRegion, bool) {
	for _, r := range regions {
		if r.array {
			continue
		}
		if len(path) == 0 && r.start == -1 {
			return r, true
		}
		if len(path) > 0 && r.start >= 0 && slices.Equal(r.path, path) {
			return r, true
		}
	}
	return tomlRegion{}, false
}

// bakeTOML writes values into the table markata-go.<table...>. It edits the
// matching [table] when present, otherwise dotted keys in an ancestor table
// that already uses them, otherwise it appends a new [table].
func bakeTOML(data []byte, table []string, values []BakeValue) ([]byte, error) {
	newline := detectNewline(data)
	lines := splitLinesKeepEnds(data)
	scanned := scanTOML(lines)
	regions := tomlRegions(scanned)
	full := append([]string{"markata-go"}, table...)

	if r, ok := findTOMLRegion(regions, full); ok {
		return tomlEditSection(lines, scanned, r.start, r.end, nil, values, newline)
	}

	for k := len(full) - 1; k >= 0; k-- {
		r, ok := findTOMLRegion(regions, full[:k])
		if !ok {
			continue
		}
		rel := full[k:]
		dotted := false
		for i := r.start + 1; i < r.end; i++ {
			key := scanned[i].key
			if key == nil {
				continue
			}
			if len(key) <= len(rel) && slices.Equal(key, rel[:len(key)]) {
				return nil, fmt.Errorf("%w: %s is not a table; convert it to a [%s] table",
					ErrBakeUnsupportedLayout, strings.Join(full[:k+len(key)], "."), strings.Join(full, "."))
			}
			if len(key) > len(rel) && slices.Equal(key[:len(rel)], rel) {
				dotted = true
			}
		}
		if dotted {
			return tomlEditSection(lines, scanned, r.start, r.end, rel, values, newline)
		}
	}

	sets := withoutRemovals(values)
	if len(sets) == 0 {
		// Nothing defines the table, so there is nothing to remove.
		return data, nil
	}
	var b strings.Builder
	b.WriteString(strings.Join(lines, ""))
	if b.Len() > 0 {
		if !strings.HasSuffix(b.String(), "\n") {
			b.WriteString(newline)
		}
		b.WriteString(newline)
	}
	b.WriteString("[" + strings.Join(full, ".") + "]" + newline)
	for _, v := range sets {
		b.WriteString(v.Key + " = " + bakeScalar(v.Value) + newline)
	}
	return []byte(b.String()), nil
}

// tomlValueSpan returns the last line of the value that starts on line i and
// the comment trailing it. ok is false for values that cannot be replaced
// safely (multi-line strings and inline tables).
func tomlValueSpan(lines []string, scanned []tomlLine, i, end int) (last int, comment string, ok bool) {
	value := strings.TrimSpace(scanned[i].value)
	if !strings.HasPrefix(value, "[") {
		comment, ok = tomlTrailingComment(value)
		return i, comment, ok
	}
	text := value
	last = i
	for open := tomlBracketDelta(value); open > 0; {
		if last+1 >= end {
			return 0, "", false
		}
		last++
		body, _ := lineEnding(lines[last])
		text += "\n" + body
		open += tomlBracketDelta(body)
	}
	depth := 0
	var quote byte
	for j := 0; j < len(text); j++ {
		c := text[j]
		if quote != 0 {
			if c == '\\' && quote == '"' {
				j++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
		case '#':
			if nl := strings.IndexByte(text[j:], '\n'); nl >= 0 {
				j += nl
				continue
			}
			return 0, "", false
		case '[', '{':
			depth++
		case ']', '}':
			depth--
			if depth == 0 {
				rest := strings.TrimSpace(text[j+1:])
				if rest != "" && !strings.HasPrefix(rest, "#") {
					return 0, "", false
				}
				return last, rest, true
			}
		}
	}
	return 0, "", false
}

func tomlEditSection(lines []string, scanned []tomlLine, start, end int, prefix []string, values []BakeValue, newline string) ([]byte, error) {
	out := append([]string{}, lines...)
	lastKey := start
	indent := ""
	found := make(map[string]bool, len(values))
	for i := start + 1; i < end; i++ {
		l := scanned[i]
		if l.key == nil {
			continue
		}
		lastKey = i
		// Include the multi-line tail of the previous value.
		for j := i + 1; j < end && scanned[j].key == nil && scanned[j].header == nil; j++ {
			if strings.TrimSpace(lines[j]) == "" || strings.HasPrefix(strings.TrimSpace(lines[j]), "#") {
				break
			}
			lastKey = j
		}
		if len(prefix) == 0 || (len(l.key) > len(prefix) && slices.Equal(l.key[:len(prefix)], prefix)) {
			indent = l.keyIndent
		}
		if len(l.key) != len(prefix)+1 || !slices.Equal(l.key[:len(prefix)], prefix) {
			continue
		}
		name := l.key[len(prefix)]
		for _, v := range values {
			if v.Key != name {
				continue
			}
			last, comment, ok := tomlValueSpan(lines, scanned, i, end)
			if !ok {
				return nil, fmt.Errorf("%w: %s is not a single-line value", ErrBakeUnsupportedLayout, name)
			}
			found[name] = true
			if IsBakeRemove(v.Value) {
				for j := i; j <= last; j++ {
					out[j] = ""
				}
				continue
			}
			_, ending := lineEnding(lines[last])
			keyText := strings.Join(append(append([]string{}, prefix...), name), ".")
			replaced := l.keyIndent + keyText + " = " + bakeScalar(v.Value)
			if comment != "" {
				replaced += " " + comment
			}
			out[i] = replaced + ending
			for j := i + 1; j <= last; j++ {
				out[j] = ""
			}
			found[name] = true
		}
	}

	var inserts []string
	for _, v := range values {
		if found[v.Key] || IsBakeRemove(v.Value) {
			continue
		}
		keyText := strings.Join(append(append([]string{}, prefix...), v.Key), ".")
		inserts = append(inserts, indent+keyText+" = "+bakeScalar(v.Value)+newline)
	}
	if len(inserts) > 0 {
		if lastKey < 0 {
			return nil, fmt.Errorf("%w: no place to insert keys", ErrBakeUnsupportedLayout)
		}
		if out[lastKey] != "" && !strings.HasSuffix(out[lastKey], "\n") {
			out[lastKey] += newline
		}
		tail := append([]string{}, out[lastKey+1:]...)
		out = append(append(out[:lastKey+1], inserts...), tail...)
	}
	return []byte(strings.Join(out, "")), nil
}

// ---------------------------------------------------------------------------
// YAML
// ---------------------------------------------------------------------------

func yamlMappingValue(mapping *yaml.Node, key string) (keyNode, valueNode *yaml.Node) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i], mapping.Content[i+1]
		}
	}
	return nil, nil
}

func yamlIsNull(node *yaml.Node) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.Tag == "!!null"
}

// yamlChildIndent returns the indentation of keys inside mapping, or
// parentCol+step when the mapping is empty.
func yamlChildIndent(mapping, keyNode *yaml.Node, step int) int {
	if mapping != nil && mapping.Kind == yaml.MappingNode && len(mapping.Content) > 0 {
		return mapping.Content[0].Column - 1
	}
	return keyNode.Column - 1 + step
}

func yamlMaxLine(node *yaml.Node) int {
	line := node.Line
	for _, child := range node.Content {
		if l := yamlMaxLine(child); l > line {
			line = l
		}
	}
	return line
}

func checkYAMLMapping(keyNode, valueNode *yaml.Node, name string) error {
	if valueNode.Kind != yaml.MappingNode && !yamlIsNull(valueNode) {
		return fmt.Errorf("%w: %s is not a mapping", ErrBakeUnsupportedLayout, name)
	}
	if valueNode.Style&yaml.FlowStyle != 0 || (valueNode.Kind == yaml.MappingNode && valueNode.Line == keyNode.Line) {
		return fmt.Errorf("%w: %s uses flow style; use a block mapping", ErrBakeUnsupportedLayout, name)
	}
	if yamlIsNull(valueNode) && strings.TrimSpace(valueNode.Value) != "" {
		return fmt.Errorf("%w: %s is an explicit null", ErrBakeUnsupportedLayout, name)
	}
	if valueNode.Alias != nil || valueNode.Kind == yaml.AliasNode || valueNode.Anchor != "" {
		return fmt.Errorf("%w: %s uses anchors or aliases", ErrBakeUnsupportedLayout, name)
	}
	return nil
}

// bakeYAML writes values into the mapping markata-go.<table...>, creating
// missing mappings below the deepest one that exists.
func bakeYAML(data []byte, table []string, values []BakeValue) ([]byte, error) {
	newline := detectNewline(data)
	lines := splitLinesKeepEnds(data)
	var doc yaml.Node
	if len(bytes.TrimSpace(data)) > 0 {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, err
		}
	}

	var rootMap *yaml.Node
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		rootMap = doc.Content[0]
	}
	if rootMap != nil && rootMap.Kind != yaml.MappingNode && !yamlIsNull(rootMap) {
		return nil, fmt.Errorf("%w: document root is not a mapping", ErrBakeUnsupportedLayout)
	}
	if rootMap != nil && rootMap.Style&yaml.FlowStyle != 0 {
		return nil, fmt.Errorf("%w: document root uses flow style", ErrBakeUnsupportedLayout)
	}

	step := 2
	markataKey, markataVal := yamlMappingValue(rootMap, "markata-go")
	if markataVal != nil && markataVal.Kind == yaml.MappingNode && len(markataVal.Content) > 0 && markataKey != nil {
		if s := markataVal.Content[0].Column - markataKey.Column; s > 0 {
			step = s
		}
	}
	pad := func(n int) string { return strings.Repeat(" ", n) }
	// nested renders mapping keys for path followed by the values, starting
	// at indent.
	nested := func(path []string, indent int, only func(string) bool) []string {
		result := make([]string, 0, len(path)+len(values))
		for i, key := range path {
			result = append(result, pad(indent+i*step)+key+":"+newline)
		}
		for _, v := range values {
			if IsBakeRemove(v.Value) {
				continue
			}
			if only == nil || only(v.Key) {
				result = append(result, pad(indent+len(path)*step)+v.Key+": "+bakeScalar(v.Value)+newline)
			}
		}
		return result
	}
	insertAfter := func(src []string, lineNo int, add []string) []byte {
		out := append([]string{}, src...)
		if lineNo > 0 && !strings.HasSuffix(out[lineNo-1], "\n") {
			out[lineNo-1] += newline
		}
		tail := append([]string{}, out[lineNo:]...)
		out = append(append(out[:lineNo], add...), tail...)
		return []byte(strings.Join(out, ""))
	}

	hasSets := len(withoutRemovals(values)) > 0
	if markataKey == nil {
		if !hasSets {
			return data, nil
		}
		var b strings.Builder
		b.WriteString(strings.Join(lines, ""))
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
			b.WriteString(newline)
		}
		for _, line := range nested(append([]string{"markata-go"}, table...), 0, nil) {
			b.WriteString(line)
		}
		return []byte(b.String()), nil
	}
	if err := checkYAMLMapping(markataKey, markataVal, "markata-go"); err != nil {
		return nil, err
	}

	parentKey, parent := markataKey, markataVal
	for depth, seg := range table {
		keyNode, valNode := yamlMappingValue(parent, seg)
		if keyNode == nil {
			if !hasSets {
				return data, nil
			}
			indent := yamlChildIndent(parent, parentKey, step)
			return insertAfter(lines, parentKey.Line, nested(table[depth:], indent, nil)), nil
		}
		name := "markata-go." + strings.Join(table[:depth+1], ".")
		if err := checkYAMLMapping(keyNode, valNode, name); err != nil {
			return nil, err
		}
		parentKey, parent = keyNode, valNode
	}

	out := append([]string{}, lines...)
	found := make(map[string]bool, len(values))
	for _, v := range values {
		keyNode, valNode := yamlMappingValue(parent, v.Key)
		if keyNode == nil {
			continue
		}
		unsupported := fmt.Errorf("%w: markata-go.%s is not a single-line value", ErrBakeUnsupportedLayout,
			strings.Join(append(append([]string{}, table...), v.Key), "."))
		if valNode.Alias != nil || valNode.Anchor != "" || valNode.Kind == yaml.AliasNode {
			return nil, unsupported
		}
		lastLine := yamlMaxLine(valNode)
		if IsBakeRemove(v.Value) {
			if valNode.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 || (valNode.Kind == yaml.ScalarNode && valNode.Line != keyNode.Line) {
				return nil, unsupported
			}
			for l := keyNode.Line - 1; l < lastLine; l++ {
				out[l] = ""
			}
			found[v.Key] = true
			continue
		}
		body, ending := lineEnding(out[keyNode.Line-1])
		runes := []rune(body)
		var replaced string
		switch {
		case valNode.Kind == yaml.ScalarNode && valNode.Line == keyNode.Line &&
			valNode.Style&(yaml.LiteralStyle|yaml.FoldedStyle) == 0:
			if valNode.Column-1 > len(runes) {
				return nil, unsupported
			}
			replaced = string(runes[:valNode.Column-1]) + bakeScalar(v.Value)
		case valNode.Kind == yaml.SequenceNode && valNode.Style&yaml.FlowStyle != 0 && lastLine == keyNode.Line:
			if valNode.Column-1 > len(runes) {
				return nil, unsupported
			}
			replaced = string(runes[:valNode.Column-1]) + bakeScalar(v.Value)
		case valNode.Kind == yaml.SequenceNode && valNode.Style&yaml.FlowStyle == 0 && valNode.Line > keyNode.Line:
			if keyNode.Style != 0 {
				return nil, unsupported
			}
			replaced = pad(keyNode.Column-1) + keyNode.Value + ": " + bakeScalar(v.Value)
		default:
			return nil, unsupported
		}
		comment := valNode.LineComment
		if comment == "" {
			comment = keyNode.LineComment
		}
		if comment != "" && valNode.Kind != yaml.SequenceNode {
			replaced += " " + comment
		}
		if valNode.Kind == yaml.SequenceNode && lastLine > keyNode.Line {
			_, ending = lineEnding(out[lastLine-1])
			for l := keyNode.Line; l < lastLine; l++ {
				out[l] = ""
			}
		}
		out[keyNode.Line-1] = replaced + ending
		found[v.Key] = true
	}

	indent := yamlChildIndent(parent, parentKey, step)
	add := nested(nil, indent, func(key string) bool { return !found[key] })
	if len(add) == 0 {
		return []byte(strings.Join(out, "")), nil
	}
	return insertAfter(out, parentKey.Line, add), nil
}

// ---------------------------------------------------------------------------
// JSON
// ---------------------------------------------------------------------------

// orderedObject keeps JSON object keys in document order.
type orderedObject struct {
	keys   []string
	values map[string]any
}

func decodeOrderedJSON(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			obj := &orderedObject{values: map[string]any{}}
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, _ := keyTok.(string)
				val, err := decodeOrderedJSON(dec)
				if err != nil {
					return nil, err
				}
				if _, exists := obj.values[key]; !exists {
					obj.keys = append(obj.keys, key)
				}
				obj.values[key] = val
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return obj, nil
		case '[':
			arr := []any{}
			for dec.More() {
				val, err := decodeOrderedJSON(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, val)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return arr, nil
		}
	}
	return tok, nil
}

func encodeOrderedJSON(b *bytes.Buffer, value any, indent string, depth int) error {
	pad := strings.Repeat(indent, depth)
	switch v := value.(type) {
	case *orderedObject:
		if len(v.keys) == 0 {
			b.WriteString("{}")
			return nil
		}
		b.WriteString("{\n")
		for i, key := range v.keys {
			keyJSON, _ := json.Marshal(key)
			b.WriteString(pad + indent)
			b.Write(keyJSON)
			b.WriteString(": ")
			if err := encodeOrderedJSON(b, v.values[key], indent, depth+1); err != nil {
				return err
			}
			if i < len(v.keys)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(pad + "}")
	case []any:
		if len(v) == 0 {
			b.WriteString("[]")
			return nil
		}
		b.WriteString("[\n")
		for i, item := range v {
			b.WriteString(pad + indent)
			if err := encodeOrderedJSON(b, item, indent, depth+1); err != nil {
				return err
			}
			if i < len(v)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(pad + "]")
	case json.Number:
		b.WriteString(v.String())
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return err
		}
		b.Write(encoded)
	}
	return nil
}

var jsonIndentPattern = regexp.MustCompile(`(?m)^([ \t]+)"`)

func bakeJSON(data []byte, table []string, values []BakeValue) ([]byte, error) {
	var root any = &orderedObject{values: map[string]any{}}
	if len(bytes.TrimSpace(data)) > 0 {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		var err error
		root, err = decodeOrderedJSON(dec)
		if err != nil {
			return nil, err
		}
	}
	rootObj, ok := root.(*orderedObject)
	if !ok {
		return nil, fmt.Errorf("%w: document root is not an object", ErrBakeUnsupportedLayout)
	}
	child := func(parent *orderedObject, key string) (*orderedObject, error) {
		existing, exists := parent.values[key]
		if !exists || existing == nil {
			obj := &orderedObject{values: map[string]any{}}
			if !exists {
				parent.keys = append(parent.keys, key)
			}
			parent.values[key] = obj
			return obj, nil
		}
		obj, ok := existing.(*orderedObject)
		if !ok {
			return nil, fmt.Errorf("%w: %s is not an object", ErrBakeUnsupportedLayout, key)
		}
		return obj, nil
	}
	if len(withoutRemovals(values)) == 0 {
		// Only removals: walk without creating tables; a missing table
		// means there is nothing to remove.
		node := rootObj
		for _, key := range append([]string{"markata-go"}, table...) {
			next, ok := node.values[key].(*orderedObject)
			if !ok {
				if _, exists := node.values[key]; exists && node.values[key] != nil {
					return nil, fmt.Errorf("%w: %s is not an object", ErrBakeUnsupportedLayout, key)
				}
				return data, nil
			}
			node = next
		}
		child = func(parent *orderedObject, key string) (*orderedObject, error) {
			return parent.values[key].(*orderedObject), nil
		}
	}
	section, err := child(rootObj, "markata-go")
	if err != nil {
		return nil, err
	}
	for _, key := range table {
		section, err = child(section, key)
		if err != nil {
			return nil, err
		}
	}
	for _, v := range values {
		if IsBakeRemove(v.Value) {
			if _, exists := section.values[v.Key]; exists {
				delete(section.values, v.Key)
				section.keys = slices.DeleteFunc(section.keys, func(k string) bool { return k == v.Key })
			}
			continue
		}
		if _, exists := section.values[v.Key]; !exists {
			section.keys = append(section.keys, v.Key)
		}
		section.values[v.Key] = v.Value
	}

	indent := "  "
	if m := jsonIndentPattern.FindSubmatch(data); m != nil {
		indent = string(m[1])
	}
	var b bytes.Buffer
	if err := encodeOrderedJSON(&b, rootObj, indent, 0); err != nil {
		return nil, err
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}
