package main

func run() uint64 {
	masks := make([]uint64, 4)
	masks[0] = uint64(0x0F)
	masks[1] = uint64(0xF0)
	masks[2] = uint64(0xFF00)
	masks[3] = uint64(0xFF0000)
	var combined uint64
	for _, m := range masks {
		combined |= m
	}
	return combined
}
