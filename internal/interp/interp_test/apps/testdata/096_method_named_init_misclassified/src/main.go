package main

import (
	"fmt"
	"testpkg/lib"
)

func entrypoint() string {
	p := &lib.Parser{Value: "hello"}
	p.Prepare()
	return fmt.Sprintf("ready=%v value=%s", p.Ready, p.Value)
}

func main() {
	fmt.Println(entrypoint())
}
