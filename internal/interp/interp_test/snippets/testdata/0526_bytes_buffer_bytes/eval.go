package main

import "bytes"

func run() int {
	var b bytes.Buffer
	b.Write([]byte{0x68, 0x69})
	out := b.Bytes()
	if len(out) == 2 && out[0] == 'h' && out[1] == 'i' {
		return 1
	}
	return 0
}
