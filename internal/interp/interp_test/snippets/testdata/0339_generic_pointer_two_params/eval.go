package main

type Pair[K, V any] struct {
	key K
	val V
}

func (p *Pair[K, V]) Set(k K, v V) {
	p.key = k
	p.val = v
}

func (p Pair[K, V]) Both() (K, V) {
	return p.key, p.val
}

func run() string {
	p := &Pair[string, int]{}
	p.Set("answer", 42)
	k, v := p.Both()
	if v != 42 {
		return "fail"
	}
	return k
}
