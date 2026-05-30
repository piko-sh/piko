package main

type Handler struct{ Run func() int }

func run() int {
	h := Handler{Run: func() int { return 42 }}
	return h.Run()
}
