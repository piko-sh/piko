package main

func produce(ch chan<- Event) {
	ch <- NumberEvent{N: 42}
	ch <- TextEvent{S: "hello"}
	ch <- NumberEvent{N: 7}
	close(ch)
}
