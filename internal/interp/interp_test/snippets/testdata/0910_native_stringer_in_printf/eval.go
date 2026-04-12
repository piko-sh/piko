package main

import "fmt"

type ID int

func (i ID) String() string {
	return fmt.Sprintf("ID#%d", int(i))
}

func run() string {
	return fmt.Sprintf("%s | %s | %s", ID(7), ID(42), ID(100))
}
