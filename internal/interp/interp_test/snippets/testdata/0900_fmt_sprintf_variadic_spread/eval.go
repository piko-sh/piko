package main

import "fmt"

func run() string {
	args := []interface{}{30, "ago"}
	return fmt.Sprintf("%d seconds %s", args...)
}
