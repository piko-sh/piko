package main

import "fmt"

func classify(value any) string {
	switch v := value.(type) {
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("signed:%v", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("unsigned:%v", v)
	case float32, float64:
		return fmt.Sprintf("float:%v", v)
	case string:
		return "string:" + v
	case bool:
		if v {
			return "bool:true"
		}
		return "bool:false"
	case nil:
		return "nil"
	default:
		return fmt.Sprintf("other:%T", v)
	}
}

type Marker struct{}

func run() string {
	result := ""
	result += classify(42) + ";"
	result += classify(int32(-7)) + ";"
	result += classify(uint8(255)) + ";"
	result += classify(3.14) + ";"
	result += classify(float32(1.5)) + ";"
	result += classify("hello") + ";"
	result += classify(true) + ";"
	result += classify(false) + ";"
	result += classify(nil) + ";"
	result += classify(Marker{})
	return result
}
