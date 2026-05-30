package main

func inner() (r int) {
	defer func() {
		if v := recover(); v != nil {
			r = 1
		}
	}()
	panic("inner")
}

func run() (result int) {
	defer func() {
		if v := recover(); v != nil {
			result = -1
		}
	}()
	result = inner()
	return
}
