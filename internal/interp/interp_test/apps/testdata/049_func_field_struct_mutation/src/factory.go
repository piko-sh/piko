package main

import "fmt"

func newGreeter(prefix string) *Greeter {
	g := &Greeter{prefix: prefix}
	g.greet = func(suffix string) string {
		g.count++
		return fmt.Sprintf("%s-%d%s", g.prefix, g.count, suffix)
	}
	return g
}
