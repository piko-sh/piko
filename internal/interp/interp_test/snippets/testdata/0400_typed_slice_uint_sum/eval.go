package main

func run() uint64 {
	values := make([]uint64, 6)
	values[0] = uint64(100)
	values[1] = uint64(200)
	values[2] = uint64(300)
	values[3] = uint64(400)
	values[4] = uint64(500)
	values[5] = uint64(600)
	var total uint64
	for _, v := range values {
		total += v
	}
	return total
}
