package main

func makeIntBox(v int) IntBox {
	return IntBox{Value: v}
}

func makeStringPair(key string, value string) Pair[string, string] {
	return Pair[string, string]{Key: key, Value: value}
}

func unboxAdd(boxes []IntBox) int {
	total := 0
	for _, b := range boxes {
		total += b.Value
	}
	return total
}
