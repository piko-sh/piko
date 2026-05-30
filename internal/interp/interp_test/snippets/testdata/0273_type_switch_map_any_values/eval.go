package main

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
