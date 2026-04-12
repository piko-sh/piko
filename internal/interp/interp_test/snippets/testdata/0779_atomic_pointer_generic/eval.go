package main

import (
	"fmt"
	"sync/atomic"
)

type Config struct {
	Name string
}

func run() string {
	var p atomic.Pointer[Config]
	c1 := &Config{Name: "alpha"}
	p.Store(c1)
	loaded := p.Load()
	result := fmt.Sprintf("loaded=%s;", loaded.Name)

	c2 := &Config{Name: "beta"}
	swapped := p.CompareAndSwap(c1, c2)
	result += fmt.Sprintf("cas=%t,now=%s;", swapped, p.Load().Name)

	c3 := &Config{Name: "gamma"}
	prev := p.Swap(c3)
	result += fmt.Sprintf("swap=%s,now=%s", prev.Name, p.Load().Name)
	return result
}
