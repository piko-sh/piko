package main

func run() int {
	z := 3 + 4i
	return int(real(z))*10 + int(imag(z))
}
