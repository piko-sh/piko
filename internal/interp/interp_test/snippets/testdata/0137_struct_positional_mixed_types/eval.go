package main

type Record struct {
	Name string
	Age  int
}

func run() int {
	r := Record{"alice", 30}
	return r.Age + len(r.Name)
}
