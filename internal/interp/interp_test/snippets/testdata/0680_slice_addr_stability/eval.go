package main

import "fmt"

func run() string {
	result := ""

	a := make([]int, 3, 3)
	a[0] = 100
	pointer := &a[0]

	grown := append(a, 4, 5, 6)

	result += fmt.Sprintf("origFirst=%d;", a[0])
	result += fmt.Sprintf("derefPointer=%d;", *pointer)
	result += fmt.Sprintf("grownFirst=%d;", grown[0])

	*pointer = 999
	result += fmt.Sprintf("afterPointerMutate_orig=%d;", a[0])
	result += fmt.Sprintf("afterPointerMutate_grown=%d;", grown[0])

	b := make([]int, 4, 8)
	b[0], b[1], b[2], b[3] = 10, 20, 30, 40
	pointer2 := &b[1]

	b[3] = 999
	result += fmt.Sprintf("stableAcrossOtherWrites=%d;", *pointer2)

	extended := append(b, 50)
	*pointer2 = 22
	result += fmt.Sprintf("inPlaceAppend_sees_mutation=%d;", extended[1])

	return result
}
