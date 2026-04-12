package main

func run() string {
	var i int32 = 42
	var a any = i
	_, ok := a.(int32)
	if ok {
		return "ok"
	}
	return "fail"
}
