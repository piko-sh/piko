package main

import "fmt"

type objectPathResult struct {
	part  string
	path  string
	pipe  string
	piped bool
	wild  bool
	more  bool
}

func parseObjectPath(path string) (r objectPathResult) {
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			r.part = path[:i]
			r.path = path[i+1:]
			r.more = true
			return
		}
	}
	r.part = path
	return
}

type Result struct {
	N int
	M string
}

type parseContext struct {
	json  string
	value Result
}

func parseObject(c *parseContext, path string) {
	rp := parseObjectPath(path)
	if rp.part == "x" {
		c.value.N = 7
		c.value.M = "found"
	}
}

func Get(json, path string) Result {
	c := &parseContext{json: json}
	parseObject(c, path)
	return c.value
}

func run() string {
	r := Get(`{"x":1}`, "x")
	return fmt.Sprintf("N=%d M=%q", r.N, r.M)
}
