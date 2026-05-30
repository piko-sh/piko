package main

func run() int {
	z := complex(3.0, 4.0)
	return int(real(z))*10 + int(imag(z))
}
