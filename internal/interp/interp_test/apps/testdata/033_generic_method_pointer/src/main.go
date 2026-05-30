package main

import "fmt"

func entrypoint() string {
	s := newStack[int]()
	s.Push(10)
	s.Push(20)
	s.Push(30)
	all := s.All()
	total := 0
	for _, v := range all {
		total += v
	}
	return fmt.Sprintf("values=%v sum=%d", all, total)
}

func main() {
	fmt.Println(entrypoint())
}
