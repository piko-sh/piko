package main

func run() string {
	var u uint32 = 42
	var a any = u
	_, ok := a.(uint32)
	if ok {
		return "ok"
	}
	return "fail"
}
