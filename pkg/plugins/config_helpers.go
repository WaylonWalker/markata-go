package plugins

import "math"

// parseIntFromInterface converts a decoded numeric configuration value to int.
// Only values that are integral and representable by the platform's int type
// are accepted. Other values are rejected instead of being truncated or
// allowed to overflow.
func parseIntFromInterface(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return intFromInt64(v)
	case uint:
		return intFromUint64(uint64(v))
	case uint8:
		return intFromUint64(uint64(v))
	case uint16:
		return intFromUint64(uint64(v))
	case uint32:
		return intFromUint64(uint64(v))
	case uint64:
		return intFromUint64(v)
	case float64:
		return intFromFloat64(v)
	case float32:
		return intFromFloat64(float64(v))
	default:
		return 0, false
	}
}

func intFromInt64(value int64) (int, bool) {
	minInt, maxInt := intBounds()
	if value < int64(minInt) || value > int64(maxInt) {
		return 0, false
	}
	return int(value), true
}

func intFromUint64(value uint64) (int, bool) {
	_, maxInt := intBounds()
	if value > uint64(maxInt) { //nolint:gosec // maxInt is the platform int upper bound.
		return 0, false
	}
	return int(value), true //nolint:gosec // value was checked against the platform int upper bound.
}

func intFromFloat64(value float64) (int, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value {
		return 0, false
	}

	minInt, maxInt := intBounds()
	// maxInt + 1 is representable as a float on every supported platform. On
	// 64-bit platforms, float64(maxInt) rounds to 2^63, so the exclusive upper
	// bound also rejects the first value that cannot fit in an int64.
	if value < float64(minInt) || value >= float64(maxInt)+1 {
		return 0, false
	}
	return int(value), true
}

func intBounds() (minInt, maxInt int) {
	maxInt = int(^uint(0) >> 1)
	return -maxInt - 1, maxInt
}
