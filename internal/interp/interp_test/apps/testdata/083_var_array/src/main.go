package main

import (
	"fmt"

	"testpkg/uuidish"
)

func hex2(b byte) string {
	const digits = "0123456789abcdef"
	return string(digits[b>>4]) + string(digits[b&0x0f])
}

func entrypoint() string {
	id := uuidish.NamespaceURL
	first := id[0] == 0x6b && id[1] == 0xa7 && id[2] == 0xb8 && id[3] == 0x11
	flag := "false"
	if first {
		flag = "true"
	}
	return "b0=" + hex2(id[0]) + " b4=" + hex2(id[4]) + " b15=" + hex2(id[15]) + " first=" + flag
}

func main() {
	fmt.Println(entrypoint())
}
