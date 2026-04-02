package main

func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, nil
	}
	return a / b, nil
}

func run() int {
	v, err := divide(10, 2)
	r := v
	if err != nil {
		r = -1
	}
	return r
}
