package main

import (
	"fmt"
	"sync/atomic"
)

type Config struct {
	Name    string
	Version int
}

func run() string {
	result := ""
	var v atomic.Value
	v.Store(Config{Name: "alpha", Version: 1})
	c1 := v.Load().(Config)
	result += fmt.Sprintf("c1=%s/%d;", c1.Name, c1.Version)

	v.Store(Config{Name: "beta", Version: 2})
	c2 := v.Load().(Config)
	result += fmt.Sprintf("c2=%s/%d;", c2.Name, c2.Version)

	swapped := v.CompareAndSwap(c2, Config{Name: "gamma", Version: 3})
	c3 := v.Load().(Config)
	result += fmt.Sprintf("cas=%t,c3=%s/%d", swapped, c3.Name, c3.Version)

	return result
}
