package main

func run() int {
	const big = 1 << 31
	x := int64(big)
	if x == 2147483648 {
		return 1
	}
	return 0
}
