package main

import "fmt"

func entrypoint() string {
	shapes := []Shape{
		circle{radius: 5},
		square{side: 4},
		triangle{base: 4, height: 3},
	}
	parts := make(map[string]float64, len(shapes))
	total := 0.0
	for _, s := range shapes {
		a := s.Area()
		parts[s.Name()] = a
		total += a
	}
	return fmt.Sprintf("circle=%.2f square=%.2f triangle=%.2f sum=%.2f",
		parts["circle"], parts["square"], parts["triangle"], total)
}

func main() {
	fmt.Println(entrypoint())
}
