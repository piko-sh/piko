package main

func tryPanic() string {
	defer func() {
		if r := recover(); r != nil {
			_ = r
		}
	}()
	panic("boom")
}

func safeCall() int {
	tryPanic()
	return 42
}

func run() int {
	return safeCall()
}
