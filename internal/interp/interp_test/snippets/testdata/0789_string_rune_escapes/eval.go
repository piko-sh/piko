package main

import "fmt"

func run() string {
	bell := "\a"
	backspace := "\b"
	form := "\f"
	vertical := "\v"
	hex := "\xAB"
	octal := "\377"
	mixed := "\a\b\f\v\xCD\077"

	accent := 'é'
	emoji := '\U0001F600'
	hexRune := '\xFF'

	return fmt.Sprintf(
		"bell=%d;bs=%d;ff=%d;vt=%d;hex=%d;oct=%d;mixed_len=%d;accent=%d;emoji=%d;hexrune=%d",
		bell[0], backspace[0], form[0], vertical[0], hex[0], octal[0], len(mixed),
		accent, emoji, hexRune)
}
