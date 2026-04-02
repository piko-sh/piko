package main

func inc(x int) int { return x + 1 }
func dec(x int) int { return x - 1 }

func run() int {
	m := map[string]func(int) int{"inc": inc, "dec": dec}
	return m["inc"](20) + m["dec"](20)
}
