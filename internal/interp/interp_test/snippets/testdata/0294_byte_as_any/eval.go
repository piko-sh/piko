package main

func run() string {
	var b byte = 0x7F
	var a any = b
	_, ok := a.(byte)
	if ok {
		return "ok"
	}
	return "fail"
}
