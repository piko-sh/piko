package main

import "fmt"

func run() string {
	result := ""

	result += string(rune('A')) + ";"
	result += string(rune(0x4E2D)) + ";"
	result += string(rune(0x1F600)) + ";"
	result += string([]rune{'h', 'i', 0x4E2D}) + ";"
	result += string([]byte{0xE4, 0xB8, 0xAD}) + ";"

	tooHigh := rune(0x110000)
	result += string(tooHigh) + ";"

	negative := rune(-1)
	result += string(negative) + ";"

	result += fmt.Sprintf("len=%d", len(string(rune(0x1F600))))

	return result
}
