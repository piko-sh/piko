package main

func makeBuffer() Buffer {
	return Buffer{data: []int{}}
}

func makeBuffersInSlice(count int) []Buffer {
	out := make([]Buffer, count)
	for i := range out {
		out[i] = Buffer{data: []int{i, i + 1}}
	}
	return out
}
