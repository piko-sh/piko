package main

import (
	"fmt"
	"unicode"
)

func run() string {
	result := ""
	samples := []rune{'A', 'a', '5', ' ', '.', '日', 'Ñ'}
	for _, r := range samples {
		result += fmt.Sprintf("%c:L=%t,D=%t,S=%t,U=%t,P=%t;",
			r,
			unicode.IsLetter(r),
			unicode.IsDigit(r),
			unicode.IsSpace(r),
			unicode.IsUpper(r),
			unicode.IsPunct(r))
	}
	result += fmt.Sprintf("up=%c,low=%c,title=%c",
		unicode.ToUpper('a'),
		unicode.ToLower('Z'),
		unicode.ToTitle('h'))
	return result
}
