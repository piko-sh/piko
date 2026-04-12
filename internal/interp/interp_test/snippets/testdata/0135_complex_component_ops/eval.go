package main

func run() int {
	z1 := 5 + 12i
	z2 := 3 + 4i
	return int(real(z1-z2))*10 + int(imag(z1-z2))
}
