package main

import (
	"fmt"
	"time"
)

func run() string {
	var caught any
	func() {
		defer func() { caught = recover() }()
		t := time.Now()
		_ = t.In(nil)
	}()
	return fmt.Sprintf("%v", caught)
}
