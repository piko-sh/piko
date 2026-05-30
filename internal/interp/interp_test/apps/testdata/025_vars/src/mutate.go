package main

func bump(label string) {
	counter++
	history = append(history, label)
}
