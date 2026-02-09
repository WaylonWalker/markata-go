package plugins

func parseIntFromInterface(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case int32:
		return int(v), true
	case float32:
		return int(v), true
	default:
		return 0, false
	}
}
