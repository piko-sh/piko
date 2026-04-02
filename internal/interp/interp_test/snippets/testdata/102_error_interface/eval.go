package main

type myError struct {
	message string
}

func (e *myError) Error() string {
	return e.message
}
func failIf(fail bool) error {
	if fail {
		return &myError{message: "oops"}
	}
	return nil
}

func run() int {
	err := failIf(true)
	r := 0
	if err != nil {
		r = 1
	}
	return r
}
