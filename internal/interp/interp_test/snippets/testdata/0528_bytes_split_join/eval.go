package main

import "bytes"

func run() string {
	parts := bytes.Split([]byte("a,b,c"), []byte(","))
	joined := bytes.Join(parts, []byte("|"))
	return string(joined)
}
