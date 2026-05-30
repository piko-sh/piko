package main

func produce(yield func(int) bool) {
	for i := 0; i < 3; i++ {
		if !yield(i) {
			return
		}
	}
}

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	for v := range produce {
		if v == 1 {
			panic("boom inside range")
		}
	}
	return 99
}
