package main

import "fmt"

func run() string {

	handler := func(v int) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("recovered: %v", r)
			}
		}()
		if v < 0 {
			panic("negative")
		}
		return nil
	}

	a := handler(5)
	b := handler(-1)
	return fmt.Sprintf("a=%v b=%v", a, b)
}
