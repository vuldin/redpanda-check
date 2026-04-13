package checks

// boolFromConfig extracts a boolean value from the admin API config map.
// Returns false if the key is missing or not a bool.
func boolFromConfig(config map[string]any, key string) bool {
	val, ok := config[key]
	if !ok {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	default:
		return false
	}
}

// stringFromConfig extracts a string value from the admin API config map.
// Returns empty string if the key is missing or not a string.
func stringFromConfig(config map[string]any, key string) string {
	val, ok := config[key]
	if !ok {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	default:
		return ""
	}
}

// intFromConfig extracts an integer value from the admin API config map.
// JSON numbers decode as float64, so we handle that case.
func intFromConfig(config map[string]any, key string) int {
	val, ok := config[key]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}
