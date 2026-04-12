package main

import "fmt"

func entrypoint() string {
	return fmt.Sprintf("kbBytes=%d mbBytes=%d gbKb=%d", bytesPerKB, bytesPerMB, kbPerGB)
}

func main() {
	fmt.Println(entrypoint())
}
