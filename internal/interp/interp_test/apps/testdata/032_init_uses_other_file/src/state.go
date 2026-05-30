package main

var events []string

func record(label string) {
	events = append(events, label)
}
