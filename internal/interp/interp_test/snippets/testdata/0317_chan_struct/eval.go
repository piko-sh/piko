package main

type Msg struct {
	Code int
	Body string
}

func run() string {
	ch := make(chan Msg, 3)
	ch <- Msg{Code: 1, Body: "a"}
	ch <- Msg{Code: 2, Body: "b"}
	close(ch)
	out := ""
	for m := range ch {
		out += string(rune('0'+m.Code)) + m.Body
	}
	return out
}
