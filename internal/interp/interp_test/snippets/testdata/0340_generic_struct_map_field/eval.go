package main

type Index[K comparable, V any] struct {
	store map[K]V
}

func newIndex[K comparable, V any]() *Index[K, V] {
	return &Index[K, V]{store: make(map[K]V)}
}

func (i *Index[K, V]) Put(k K, v V) {
	i.store[k] = v
}

func (i *Index[K, V]) Get(k K) V {
	return i.store[k]
}

func run() int {
	idx := newIndex[string, int]()
	idx.Put("answer", 42)
	idx.Put("year", 2026)
	return idx.Get("answer") + idx.Get("year")
}
