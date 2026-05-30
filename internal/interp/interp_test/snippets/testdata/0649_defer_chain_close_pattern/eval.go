package main

type resource struct {
	name   string
	closed bool
}

func (r *resource) Close() error {
	r.closed = true
	return nil
}

func run() int {
	r1 := &resource{name: "a"}
	defer r1.Close()
	r2 := &resource{name: "b"}
	defer r2.Close()
	return 0
}
