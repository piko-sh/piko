package main

import "fmt"

func run() string {
	result := ""

	maxRune := string(rune(0x10FFFF))
	result += fmt.Sprintf("max:len=%d/bytes=%v;", len(maxRune), []byte(maxRune))

	surrogate := string(rune(0xD800))
	result += fmt.Sprintf("surr:len=%d/bytes=%v;", len(surrogate), []byte(surrogate))

	negative := string(rune(-1))
	result += fmt.Sprintf("neg:len=%d/bytes=%v;", len(negative), []byte(negative))

	tooBig := string(rune(0x110000))
	result += fmt.Sprintf("big:len=%d/bytes=%v;", len(tooBig), []byte(tooBig))

	plain := string(rune('A'))
	result += fmt.Sprintf("A:len=%d/%q", len(plain), plain)

	return result
}
