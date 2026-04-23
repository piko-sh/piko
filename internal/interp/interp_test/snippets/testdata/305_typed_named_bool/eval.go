package main

type Flag bool

func run() string {
	var f Flag = true
	var a any = f
	_, ok := a.(Flag)
	if ok {
		return "ok"
	}
	return "fail"
}
