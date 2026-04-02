package main

// Regression test: type assertion on nil interface values from maps.
// Ensures that the interface unwrap in handleTypeAssert skips nil
// values and the ok flag is set correctly.

func run() string {
	m := map[string]any{
		"present": "value",
		"nil_val": nil,
	}

	result := ""

	// Test non-nil value assertion
	if v, ok := m["present"].(string); ok {
		result += "ok:" + v + ","
	} else {
		result += "fail,"
	}

	// Test nil value assertion (should not match string)
	if _, ok := m["nil_val"].(string); ok {
		result += "nil_matched,"
	} else {
		result += "nil_no_match,"
	}

	// Test missing key (zero value of any is nil)
	if _, ok := m["missing"].(string); ok {
		result += "missing_matched,"
	} else {
		result += "missing_no_match,"
	}

	return result
}
