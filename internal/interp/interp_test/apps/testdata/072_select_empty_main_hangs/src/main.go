package main

import "fmt"

func entrypoint() string {
	select {}
}

func main() {
	fmt.Println(entrypoint())
}
