package main

import "fmt"

func tag(v any) string {
	switch v.(type) {
	case Circle:
		return "circle=area"
	case Square:
		return "square=area"
	case Triangle:
		return "triangle=area"
	default:
		return "unknown=skip"
	}
}

func entrypoint() string {
	values := []any{Circle{R: 1}, Square{S: 2}, Triangle{B: 3, H: 4}, 42}
	tags := make([]string, len(values))
	for i, v := range values {
		tags[i] = tag(v)
	}
	return fmt.Sprintf("%s %s %s %s", tags[0], tags[1], tags[2], tags[3])
}

func main() {
	fmt.Println(entrypoint())
}
