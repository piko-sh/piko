package main

import (
	_ "encoding/json"
	f "fmt"
	. "strings"
)

func run() string {
	upper := ToUpper("hello")
	count := Count("banana", "an")
	return f.Sprintf("upper=%s,count=%d", upper, count)
}
