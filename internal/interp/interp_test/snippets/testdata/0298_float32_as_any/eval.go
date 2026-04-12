package main

func run() string {
	var f float32 = 1.5
	var a any = f
	_, ok := a.(float32)
	if ok {
		return "ok"
	}
	return "fail"
}
