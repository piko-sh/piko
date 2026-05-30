package main

func run() int {
	buffer := make([]byte, 0, 16)
	buffer = append(buffer, 'a')
	word := "bcd"
	buffer = append(buffer, word...)
	buffer = append(buffer, 'e')
	return len(buffer)
}
