package plugins

import (
	"math"
	"testing"
)

func TestParseIntFromInterface(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	maxIntAsUint64 := uint64(maxInt) //nolint:gosec // Test value is the platform int upper bound.

	tests := []struct {
		name  string
		value interface{}
		want  int
		ok    bool
	}{
		{name: "int", value: int(321), want: 321, ok: true},
		{name: "int8", value: int8(-8), want: -8, ok: true},
		{name: "int16", value: int16(16), want: 16, ok: true},
		{name: "int32", value: int32(-32), want: -32, ok: true},
		{name: "int64", value: int64(64), want: 64, ok: true},
		{name: "uint", value: uint(1), want: 1, ok: true},
		{name: "uint8", value: uint8(8), want: 8, ok: true},
		{name: "uint16", value: uint16(16), want: 16, ok: true},
		{name: "uint32", value: uint32(32), want: 32, ok: true},
		{name: "uint64", value: uint64(64), want: 64, ok: true},
		{name: "float32 integral", value: float32(32), want: 32, ok: true},
		{name: "float64 integral", value: float64(64), want: 64, ok: true},
		{name: "minimum int", value: minInt, want: minInt, ok: true},
		{name: "maximum int", value: maxInt, want: maxInt, ok: true},
		{name: "uint64 maximum int", value: maxIntAsUint64, want: maxInt, ok: true},
		{name: "fractional float64", value: 200.5, ok: false},
		{name: "fractional float32", value: float32(200.5), ok: false},
		{name: "NaN", value: math.NaN(), ok: false},
		{name: "positive infinity", value: math.Inf(1), ok: false},
		{name: "negative infinity", value: math.Inf(-1), ok: false},
		{name: "uint overflow", value: maxIntAsUint64 + 1, ok: false},
		{name: "float64 above maximum", value: float64(maxInt) + 1024, ok: false},
		{name: "float64 below minimum", value: float64(minInt) - 2048, ok: false},
		{name: "string", value: "321", ok: false},
		{name: "bool", value: true, ok: false},
		{name: "nil", value: nil, ok: false},
		{name: "unsupported object", value: struct{}{}, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseIntFromInterface(tt.value)
			if ok != tt.ok {
				t.Fatalf("parseIntFromInterface(%#v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("parseIntFromInterface(%#v) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}
