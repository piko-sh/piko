package main

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	var x any = 42
	switch v := x.(type) {
	case int:
		_ = v
		panic("inside-case boom")
	case string:
		_ = v
	}
	return 99
}
