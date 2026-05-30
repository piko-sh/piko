package main

func run() string {
	s := []interface{}{"initial"}
	s[0] = "hello"
	return s[0].(string)
}
