package main

import "sync"

func run() int {
	p := sync.Pool{
		New: func() interface{} { return 42 },
	}
	v := p.Get()
	iv, ok := v.(int)
	if !ok {
		return 0
	}
	p.Put(iv)
	v2 := p.Get()
	iv2, ok2 := v2.(int)
	if !ok2 || iv2 != iv {
		return iv
	}
	return iv2
}
