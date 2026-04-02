package main

import "fmt"

func entrypoint() int {
	r := Rect{W: 6, H: 7}
	t := Triangle{Base: 12, Height: 7}
	return rectArea(r) + triangleArea(t)
}

func main() {
	fmt.Println(entrypoint())
}
