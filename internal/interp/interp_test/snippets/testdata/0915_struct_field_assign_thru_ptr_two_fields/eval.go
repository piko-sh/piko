package main

import "fmt"

type Result struct {
	N int
	M int
}

type parseContext struct {
	json  string
	value Result
}

func parseObject(c *parseContext, i int) {
	if i > 0 {
		c.value.N = 7
		c.value.M = 99
	}
}

func Get(json string) Result {
	var i int
	var c = &parseContext{json: json}
	for ; i < len(c.json); i++ {
		if c.json[i] == '{' {
			parseObject(c, i+1)
			break
		}
	}
	return c.value
}

func run() string {
	r := Get(`{"x":1}`)
	return fmt.Sprintf("N=%d M=%d", r.N, r.M)
}
