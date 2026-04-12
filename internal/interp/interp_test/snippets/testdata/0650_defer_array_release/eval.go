package main

func run() int {
	released := 0
	for i := 0; i < 3; i++ {
		func() {
			defer func() {
				released++
			}()
		}()
	}
	return released
}
