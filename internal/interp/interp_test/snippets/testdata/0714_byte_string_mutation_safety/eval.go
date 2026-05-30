package main

import "fmt"

func run() string {
	result := ""

	s := "hello"
	b := []byte(s)
	b[0] = 'H'
	result += fmt.Sprintf("origS=%q,mutatedB=%q;", s, string(b))

	bytes := []byte{'a', 'b', 'c'}
	str := string(bytes)
	bytes[0] = 'X'
	result += fmt.Sprintf("origStr=%q,mutatedB=%q;", str, string(bytes))

	src := "world"
	copy1 := []byte(src)
	copy2 := []byte(src)
	copy1[0] = 'W'
	copy2[1] = 'O'
	result += fmt.Sprintf("origSrc=%q,c1=%q,c2=%q;", src, string(copy1), string(copy2))

	original := []byte("abc")
	roundTrip := []byte(string(original))
	roundTrip[0] = 'Z'
	result += fmt.Sprintf("orig=%q,trip=%q", string(original), string(roundTrip))

	return result
}
