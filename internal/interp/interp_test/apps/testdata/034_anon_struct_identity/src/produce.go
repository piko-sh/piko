package main

func produce() struct {
	Name string
	Age  int
} {
	return struct {
		Name string
		Age  int
	}{Name: "Bob", Age: 30}
}
