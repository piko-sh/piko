package main

import "sync"

func run() int {
	var m sync.Map
	m.Store("a", 1)
	m.Store("b", 2)
	v, ok := m.Load("a")
	if !ok {
		return 0
	}
	iv, ok := v.(int)
	if !ok || iv != 1 {
		return 0
	}
	m.Delete("a")
	if _, ok := m.Load("a"); ok {
		return 0
	}
	return 1
}
