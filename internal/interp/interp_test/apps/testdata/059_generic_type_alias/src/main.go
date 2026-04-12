package main

import "fmt"

func entrypoint() string {
	boxes := []IntBox{makeIntBox(1), makeIntBox(2), makeIntBox(3)}
	total := unboxAdd(boxes)
	pair := makeStringPair("name", "piko")
	return fmt.Sprintf("total=%d pair=%s:%s", total, pair.Key, pair.Value)
}

func main() {
	fmt.Println(entrypoint())
}
