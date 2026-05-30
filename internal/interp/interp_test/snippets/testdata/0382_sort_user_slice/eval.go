package main

import "sort"

type Person struct {
	Name string
	Age  int
}

func run() string {
	people := []Person{
		{Name: "Charlie", Age: 30},
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 28},
	}
	sort.Slice(people, func(i, j int) bool {
		return people[i].Age < people[j].Age
	})
	return people[0].Name + "," + people[1].Name + "," + people[2].Name
}
