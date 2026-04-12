package main

type Greeter struct {
	prefix string
	count  int
	greet  func(string) string
}
