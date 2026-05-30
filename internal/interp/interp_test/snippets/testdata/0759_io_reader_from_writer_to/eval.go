package main

import (
	"bytes"
	"fmt"
	"io"
)

type CountingWriter struct {
	count int64
}

func (w *CountingWriter) Write(p []byte) (int, error) {
	w.count += int64(len(p))
	return len(p), nil
}

func (w *CountingWriter) ReadFrom(r io.Reader) (int64, error) {
	buf := make([]byte, 32)
	total := int64(0)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			total += int64(n)
		}
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}

func run() string {
	src := bytes.NewBufferString("hello world example data")
	cw := &CountingWriter{}
	var rf io.ReaderFrom = cw
	read, _ := rf.ReadFrom(src)
	return fmt.Sprintf("read=%d,internal_count=%d", read, cw.count)
}
