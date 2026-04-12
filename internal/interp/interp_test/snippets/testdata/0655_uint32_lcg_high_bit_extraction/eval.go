package main

func run() int {
	state := uint32(0xDEADBEEF)
	accumulator := 0
	for index := 0; index < 16; index++ {
		state = (state*1664525 + 1013904223) & 0xFFFFFFFF
		opKind := (state >> 28) & 0xF
		id := (state >> 4) & 0x7FFF
		accumulator = accumulator*31 + int(opKind)*100000 + int(id)
	}
	return accumulator
}
