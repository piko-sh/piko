package main

type Celsius float64
type Fahrenheit float64

func toF(c Celsius) Fahrenheit {
	return Fahrenheit(c*9/5 + 32)
}

func run() int {
	return int(toF(100))
}
