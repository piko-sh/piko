package main

type MyErr struct{ N int }

func (e *MyErr) Error() string { return "x" }

func mayFail(fail bool) error {
	var p *MyErr
	if fail {
		p = &MyErr{N: 1}
	}
	return p
}

func run() int {
	e := mayFail(false)
	if e == nil {
		return 0
	}
	return 1
}
