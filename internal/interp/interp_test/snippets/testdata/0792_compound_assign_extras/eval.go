package main

import "fmt"

func run() string {
	a := uint32(0xFF00)
	a >>= 4

	b := uint32(0xFFFF)
	b &^= 0x000F

	c := 1024
	c >>= 2

	d := uint8(0xAB)
	d &^= 0x0F

	return fmt.Sprintf("a=%d,b=%d,c=%d,d=%d", a, b, c, d)
}
