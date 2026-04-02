package main

func safeDivide(a, b int) (int, error) {
	if b == 0 {
		return 0, &mathError{message: "division by zero"}
	}
	return a / b, nil
}
