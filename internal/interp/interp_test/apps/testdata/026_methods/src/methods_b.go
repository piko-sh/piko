package main

func (p person) label() string {
	if p.age >= 18 {
		return "adult-" + p.name
	}
	return "minor-" + p.name
}
