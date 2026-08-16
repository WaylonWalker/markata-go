package contentindex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// Parse decodes any released Content Index generation into the current model.
// Unknown object fields are ignored. Future generations are rejected rather
// than being silently interpreted as an older format.
func Parse(data []byte) (Index, error) {
	var envelope struct {
		Schema        *string         `json:"schema"`
		SchemaVersion json.RawMessage `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Index{}, fmt.Errorf("malformed content index JSON: %w", err)
	}
	if envelope.Schema == nil || *envelope.Schema == "" {
		return Index{}, ErrMissingSchema
	}
	if *envelope.Schema != Schema {
		return Index{}, fmt.Errorf("%w %q", ErrUnsupportedSchema, *envelope.Schema)
	}
	if envelope.SchemaVersion == nil {
		return Index{}, ErrMissingVersion
	}
	if !isJSONNumber(envelope.SchemaVersion) {
		return Index{}, fmt.Errorf("schema_version must be a JSON number")
	}
	version, err := jsonInteger(json.Number(string(envelope.SchemaVersion)))
	if err != nil {
		return Index{}, fmt.Errorf("schema_version must be an integer: %w", err)
	}
	switch version {
	case 1:
		return decodeV1(data)
	default:
		return Index{}, fmt.Errorf("%w %d (latest supported: %d)", ErrUnsupportedVersion, version, CurrentVersion)
	}
}

func isJSONNumber(value []byte) bool {
	value = bytes.TrimSpace(value)
	return len(value) > 0 && value[0] != '"' && !bytes.Equal(value, []byte("null")) && json.Unmarshal(value, new(json.Number)) == nil
}

func jsonInteger(value json.Number) (int, error) {
	if integer, err := strconv.Atoi(value.String()); err == nil {
		return integer, nil
	}
	rational, ok := new(big.Rat).SetString(value.String())
	if !ok {
		rational, ok = decimalRational(value.String())
	}
	if !ok || !rational.IsInt() {
		return 0, fmt.Errorf("%q is not an integer", value)
	}
	integer := rational.Num()
	if !integer.IsInt64() || (strconv.IntSize == 32 && (integer.Int64() < -1<<31 || integer.Int64() > 1<<31-1)) {
		return 0, fmt.Errorf("%q is outside the platform integer range", value)
	}
	return int(integer.Int64()), nil
}

func decimalRational(value string) (*big.Rat, bool) {
	parts := strings.Split(strings.ToLower(value), "e")
	if len(parts) > 2 {
		return nil, false
	}
	mantissa := parts[0]
	exponent := 0
	if len(parts) == 2 {
		var err error
		exponent, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, false
		}
	}
	point := strings.IndexByte(mantissa, '.')
	if point >= 0 {
		exponent -= len(mantissa) - point - 1
		mantissa = strings.ReplaceAll(mantissa, ".", "")
	}
	if mantissa == "" || strings.Trim(mantissa, "+-0123456789") != "" {
		return nil, false
	}
	negative := strings.HasPrefix(mantissa, "-")
	mantissa = strings.TrimPrefix(strings.TrimPrefix(mantissa, "+"), "-")
	numerator, ok := new(big.Int).SetString(mantissa, 10)
	if !ok {
		return nil, false
	}
	if negative {
		numerator.Neg(numerator)
	}
	if exponent >= 0 {
		numerator.Mul(numerator, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil))
		return new(big.Rat).SetInt(numerator), true
	}
	denominator := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-exponent)), nil)
	return new(big.Rat).SetFrac(numerator, denominator), true
}

// Marshal emits the newest supported compact JSON representation.
func Marshal(index Index) ([]byte, error) {
	if index.Schema != "" && index.Schema != Schema {
		return nil, fmt.Errorf("%w %q", ErrUnsupportedSchema, index.Schema)
	}
	if index.SchemaVersion != 0 && index.SchemaVersion != CurrentVersion {
		return nil, fmt.Errorf("%w %d", ErrUnsupportedVersion, index.SchemaVersion)
	}
	return encodeV1(index)
}

// ValidJSON reports whether data is a valid supported Content Index artifact.
func ValidJSON(data []byte) bool {
	return json.Valid(bytes.TrimSpace(data)) && func() bool { _, err := Parse(data); return err == nil }()
}
