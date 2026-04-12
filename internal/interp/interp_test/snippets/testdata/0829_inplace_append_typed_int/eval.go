package main

func run() string {
	values := make([]int, 0, 16)
	for index := 0; index < 8; index++ {
		values = append(values, index*index)
	}
	if len(values) != 8 {
		return "wrong length"
	}
	for index := 0; index < 8; index++ {
		if values[index] != index*index {
			return "wrong value"
		}
	}
	values = append(values, -1)
	values = append(values, -2)
	if values[8] != -1 || values[9] != -2 {
		return "tail wrong"
	}
	total := 0
	for index := 0; index < len(values); index++ {
		total += values[index]
	}
	if total != 0+1+4+9+16+25+36+49-1-2 {
		return "sum wrong"
	}
	return "ok"
}
