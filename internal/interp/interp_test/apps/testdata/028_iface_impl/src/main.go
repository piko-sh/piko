package main

import "fmt"

func describe(s shape) string {
	return fmt.Sprintf("%s area=%.2f", s.name(), s.area())
}

func entrypoint() string {
	shapes := []shape{circle{radius: 5}, square{side: 4}}
	parts := make([]string, 0, len(shapes))
	total := 0.0
	for _, s := range shapes {
		parts = append(parts, describe(s))
		total += s.area()
	}
	return fmt.Sprintf("%s %s total=%.2f", parts[0], parts[1], total)
}

func main() {
	fmt.Println(entrypoint())
}
