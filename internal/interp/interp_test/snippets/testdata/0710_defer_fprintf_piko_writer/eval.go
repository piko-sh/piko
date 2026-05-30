package main

import "fmt"

type collector struct {
	data []byte
}

func (c *collector) Write(p []byte) (int, error) {
	c.data = append(c.data, p...)
	return len(p), nil
}

func deferToWriter() string {
	var w collector
	for i := 0; i < 3; i++ {
		defer fmt.Fprintf(&w, "i=%d;", i)
	}
	return string(w.data)
}

func run() string {
	immediate := deferToWriter()
	return "ret=" + immediate
}
