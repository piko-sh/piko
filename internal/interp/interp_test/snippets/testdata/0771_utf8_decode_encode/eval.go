package main

import (
	"fmt"
	"unicode/utf8"
)

func run() string {
	result := ""

	s := "héllo日"
	result += fmt.Sprintf("count=%d,bytes=%d;", utf8.RuneCountInString(s), len(s))

	r, size := utf8.DecodeRuneInString(s)
	result += fmt.Sprintf("decode_str:r=%c,size=%d;", r, size)

	bytes := []byte(s)
	r2, size2 := utf8.DecodeRune(bytes)
	result += fmt.Sprintf("decode_bytes:r=%c,size=%d;", r2, size2)

	buf := make([]byte, 4)
	n := utf8.EncodeRune(buf, '日')
	result += fmt.Sprintf("encode=%d;", n)

	result += fmt.Sprintf("rune_len_A=%d,rune_len_jp=%d;",
		utf8.RuneLen('A'), utf8.RuneLen('日'))

	result += fmt.Sprintf("valid=%t,invalid=%t",
		utf8.ValidString("hello"),
		utf8.ValidString(string([]byte{0xFF, 0xFE})))

	return result
}
