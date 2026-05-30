package main

type Buffer struct {
	data []int
}

func (b *Buffer) Append(v int) {
	b.data = append(b.data, v)
}

func (b *Buffer) Sum() int {
	total := 0
	for _, v := range b.data {
		total += v
	}
	return total
}
