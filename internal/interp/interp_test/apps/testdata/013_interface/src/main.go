package main

import (
	"fmt"
	"testpkg/greeter"
)

type Speaker interface {
	Speak(message string) string
}

func announce(s Speaker, message string) string {
	return s.Speak(message)
}

func entrypoint() string {
	l := greeter.NewLoud()
	q := greeter.NewQuiet()
	return fmt.Sprintf("loud:%s quiet:%s", announce(l, "hello"), announce(q, "hello"))
}

func main() {
	fmt.Println(entrypoint())
}
