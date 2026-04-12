package main

func run() string {
	var a any = 42
	_, ok := a.(int)
	if ok {
		return "ok"
	}
	return "fail"
}
