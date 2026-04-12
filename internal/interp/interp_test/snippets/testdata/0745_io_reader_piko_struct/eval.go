package main

import (
	"fmt"
	"io"
)

type ChunkReader struct {
	data []byte
	pos  int
}

func (r *ChunkReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func run() string {
	result := ""

	r := &ChunkReader{data: []byte("hello world")}
	all, err := io.ReadAll(r)
	result += fmt.Sprintf("all=%q,err=%v;", string(all), err)

	r2 := &ChunkReader{data: []byte("abcdefghij")}
	buf := make([]byte, 3)
	for {
		n, e := r2.Read(buf)
		result += fmt.Sprintf("r=%d/%q,", n, string(buf[:n]))
		if e != nil {
			result += fmt.Sprintf("eof=%t", e == io.EOF)
			break
		}
	}
	return result
}
