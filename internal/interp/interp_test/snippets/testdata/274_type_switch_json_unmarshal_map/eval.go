package main

import "encoding/json"

// Regression test: type switch on values from json.Unmarshal into
// map[string]any. This is the exact pattern that fails when
// handleTypeAssert does not unwrap interface-typed reflect.Values
// returned by MapIndex.

func run() string {
	data := []byte(`{"name":"Alice","age":30,"active":true,"tags":["a","b"]}`)
	var m map[string]any
	_ = json.Unmarshal(data, &m)

	result := ""
	for _, key := range []string{"name", "age", "active", "tags"} {
		val, ok := m[key]
		if !ok {
			result += "missing,"
			continue
		}
		switch v := val.(type) {
		case string:
			result += "s:" + v + ","
		case float64:
			result += "f,"
		case bool:
			if v {
				result += "b:true,"
			} else {
				result += "b:false,"
			}
		case []any:
			result += "slice,"
			_ = v
		default:
			result += "default,"
		}
	}
	return result
}
