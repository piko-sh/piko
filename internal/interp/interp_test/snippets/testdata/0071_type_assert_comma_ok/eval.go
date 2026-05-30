package main

var x interface{} = 42

func run() int {
	v, ok1 := x.(int)
	_, ok2 := x.(string)
	r := v
	if !ok1 {
		r += 100
	}
	if ok2 {
		r += 200
	}
	return r
}
