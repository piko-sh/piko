package main

import "fmt"

func entrypoint() string {
	return fmt.Sprintf("result=%d", sevenTimes(9))
}

func main() {
	fmt.Println(entrypoint())
}
