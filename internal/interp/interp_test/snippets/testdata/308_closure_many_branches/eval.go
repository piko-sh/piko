package main

func run() string {
	build := func(n int) []any {
		out := []any{}
		for i := 0; i < n; i++ {
			out = append(out, i != 0)
			out = append(out, i == 0)
			if i%2 == 0 {
				out = append(out, i > 0)
			} else {
				out = append(out, i < 10)
			}
		}
		return out
	}
	parts := build(4)
	if len(parts) != 12 {
		return "wrong length"
	}
	for idx, p := range parts {
		if _, ok := p.(bool); !ok {
			return "assert fail at " + itoa(idx)
		}
	}
	return "ok"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
