package main

import (
	"fmt"
	"strings"
)

func run() string {
	result := ""

	result += fmt.Sprintf("split:%v;", strings.Split("a,,b,c,", ","))

	result += fmt.Sprintf("splitN:%v;", strings.SplitN("a:b:c:d", ":", 2))

	result += fmt.Sprintf("trim:%q;", strings.Trim("  \t hello \r\n", " \t\r\n"))

	result += fmt.Sprintf("trimFunc:%q;", strings.TrimFunc("123abc456", func(r rune) bool {
		return r >= '0' && r <= '9'
	}))

	result += fmt.Sprintf("indexByte:%d;", strings.IndexByte("hello", 'l'))
	result += fmt.Sprintf("indexByteNot:%d;", strings.IndexByte("hello", 'z'))

	result += fmt.Sprintf("repeat:%q;", strings.Repeat("ab", 3))
	result += fmt.Sprintf("repeatZero:%q;", strings.Repeat("ab", 0))

	result += fmt.Sprintf("replaceAll:%q;", strings.ReplaceAll("banana", "an", "AN"))

	before, after, found := strings.Cut("key=value", "=")
	result += fmt.Sprintf("cut:%q/%q/%v;", before, after, found)

	before2, after2, found2 := strings.Cut("nodelim", "=")
	result += fmt.Sprintf("cutNo:%q/%q/%v;", before2, after2, found2)

	result += fmt.Sprintf("contains:%v;", strings.Contains("hello", "ell"))

	result += fmt.Sprintf("eqFold:%v;", strings.EqualFold("HELLO", "hello"))

	result += fmt.Sprintf("toLower:%q;", strings.ToLower("ÜBER"))
	result += fmt.Sprintf("toUpper:%q", strings.ToUpper("hello"))

	return result
}
