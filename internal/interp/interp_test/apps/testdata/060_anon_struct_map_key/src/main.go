package main

import "fmt"

func entrypoint() string {
	register("piko.sh", "/", 10)
	register("piko.sh", "/about", 5)
	register("example.com", "/", 3)

	root, rootOk := lookup("piko.sh", "/")
	about, aboutOk := lookup("piko.sh", "/about")
	missing, missingOk := lookup("piko.sh", "/contact")

	return fmt.Sprintf("root=%d:%v about=%d:%v missing=%d:%v size=%d",
		root, rootOk, about, aboutOk, missing, missingOk, len(registry))
}

func main() {
	fmt.Println(entrypoint())
}
