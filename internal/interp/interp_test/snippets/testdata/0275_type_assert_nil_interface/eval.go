package main

func run() string {
	m := map[string]any{
		"present": "value",
		"nil_val": nil,
	}

	result := ""

	if v, ok := m["present"].(string); ok {
		result += "ok:" + v + ","
	} else {
		result += "fail,"
	}

	if _, ok := m["nil_val"].(string); ok {
		result += "nil_matched,"
	} else {
		result += "nil_no_match,"
	}

	if _, ok := m["missing"].(string); ok {
		result += "missing_matched,"
	} else {
		result += "missing_no_match,"
	}

	return result
}
