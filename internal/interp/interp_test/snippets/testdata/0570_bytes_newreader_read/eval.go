package main

import "bytes"

func run() string {
	r := bytes.NewReader([]byte("hello"))
	buf := make([]byte, 5)
	n, _ := r.Read(buf)
	return string(buf[:n])
}
