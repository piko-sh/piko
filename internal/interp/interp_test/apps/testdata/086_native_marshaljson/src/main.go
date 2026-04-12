package main

import (
	"encoding/json"
	"fmt"

	"testpkg/lib"
)

func entrypoint() string {
	out, err := json.Marshal(lib.NewLabel("alpha"))
	if err != nil {
		return "ERR " + err.Error()
	}
	return string(out)
}

func main() {
	fmt.Println(entrypoint())
}
