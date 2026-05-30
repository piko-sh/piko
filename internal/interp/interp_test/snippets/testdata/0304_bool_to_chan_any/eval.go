package main

func run() string {
	ch := make(chan any, 1)
	s := "hello"
	ch <- s != ""
	close(ch)
	v, ok := <-ch
	if !ok {
		return "closed early"
	}
	b, bOk := v.(bool)
	if !bOk {
		return "type assertion failed"
	}
	if b {
		return "ok"
	}
	return "fail"
}
