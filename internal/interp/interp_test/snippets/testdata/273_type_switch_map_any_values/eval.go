package main

// Regression test: type switch on values retrieved from map[string]any.
// MapIndex returns reflect.Value with Kind==Interface; handleTypeAssert
// must unwrap the interface before comparing the dynamic type.

func run() string {
	m := map[string]any{
		"s": "hello",
		"n": 42.0,
		"b": true,
	}

	result := ""
	for _, key := range []string{"s", "n", "b"} {
		val := m[key]
		switch v := val.(type) {
		case string:
			result += "string:" + v + ","
		case float64:
			result += "float64,"
		case bool:
			if v {
				result += "bool:true,"
			}
		default:
			result += "default,"
		}
	}
	return result
}
