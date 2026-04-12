package main

func consume(p struct {
	Name string
	Age  int
}) (string, int) {
	return p.Name, p.Age
}
