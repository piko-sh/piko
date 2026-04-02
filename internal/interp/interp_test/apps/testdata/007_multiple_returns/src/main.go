package main

import "fmt"

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune(48+n%10)) + s
		n = n / 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func entrypoint() string {
	q, r := divmod(17, 5)
	result := itoa(17) + "/" + itoa(5) + "=" + itoa(q) + "r" + itoa(r)

	_, ok := safeDivide(10, 0)
	if !ok {
		result += " div0:safe"
	}

	v2, ok2 := safeDivide(10, 3)
	if ok2 {
		result += " ok:" + itoa(v2)
	}

	return result
}

func main() {
	fmt.Println(entrypoint())
}
