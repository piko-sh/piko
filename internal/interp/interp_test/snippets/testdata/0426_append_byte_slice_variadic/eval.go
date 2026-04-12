package main

func makeSlice() []byte {
	return []byte{'h', 'i'}
}

func run() int {
	pieces := make([]byte, 0, 16)
	pieces = append(pieces, 'a')
	pieces = append(pieces, makeSlice()...)
	pieces = append(pieces, 'z')
	return len(pieces)
}
