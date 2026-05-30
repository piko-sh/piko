package main

import "fmt"

func entrypoint() string {
	c := Counter{value: 5}
	readExpr := Counter.Read
	bumpExpr := (*Counter).Bump

	initial := applyRead(readExpr, c)
	applyBump(bumpExpr, &c, 3)
	final := applyRead(readExpr, c)

	return fmt.Sprintf("initial=%d final=%d", initial, final)
}

func main() {
	fmt.Println(entrypoint())
}
