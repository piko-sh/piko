package main

func run() string {
	m := map[int]int{1: 10, 2: 20, 3: 30, 4: 40}

	visited := []int{}
	for key := range m {
		visited = append(visited, key)
		delete(m, key)
	}

	for i := 0; i < len(visited); i++ {
		for j := i + 1; j < len(visited); j++ {
			if visited[i] > visited[j] {
				visited[i], visited[j] = visited[j], visited[i]
			}
		}
	}

	result := ""
	for _, key := range visited {
		result += intToStr(int64(key)) + ","
	}
	result += "size=" + intToStr(int64(len(m)))
	return result
}

func intToStr(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
