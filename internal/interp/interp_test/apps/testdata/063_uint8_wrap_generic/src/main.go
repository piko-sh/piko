package main

import "fmt"

func entrypoint() string {
	byteWrap := wrapAdd[Byte](255, 1)
	byteOverflow := wrapAdd[Byte](200, 100)
	byteUnder := wrapSub[Byte](0, 1)

	wordWrap := wrapAdd[Word](65535, 1)
	wordOverflow := wrapAdd[Word](40000, 30000)

	signedWrap := wrapAdd[Signed](127, 1)
	signedUnder := wrapSub[Signed](-128, 1)

	return fmt.Sprintf(
		"byteWrap=%d byteOverflow=%d byteUnder=%d wordWrap=%d wordOverflow=%d signedWrap=%d signedUnder=%d",
		byteWrap, byteOverflow, byteUnder, wordWrap, wordOverflow, signedWrap, signedUnder,
	)
}

func main() {
	fmt.Println(entrypoint())
}
