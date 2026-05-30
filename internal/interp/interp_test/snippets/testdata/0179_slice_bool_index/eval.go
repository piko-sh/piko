package main

func run() int {
	s := []bool{true, false, true}
	s[1] = true
	r := 0
	for _, v := range s {
		if v {
			r++
		}
	}
	return r
}
