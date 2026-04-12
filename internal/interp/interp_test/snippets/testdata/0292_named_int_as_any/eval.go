package main

type Duration int64

func run() string {
	var d Duration = 42
	var a any = d
	_, ok := a.(Duration)
	if ok {
		return "ok"
	}
	return "fail"
}
