package main

var result string

func init() { result = "a" }
func init() { result += "b" }
func init() { result += "c" }

func run() string {
	return result
}
