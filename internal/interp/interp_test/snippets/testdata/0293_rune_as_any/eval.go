package main

func run() string {
	var r rune = 'A'
	var a any = r
	_, ok := a.(rune)
	if ok {
		return "ok"
	}
	return "fail"
}
