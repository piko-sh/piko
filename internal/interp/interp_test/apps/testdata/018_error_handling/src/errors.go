package main

type mathError struct {
	message string
}

func (e *mathError) Error() string {
	return e.message
}
