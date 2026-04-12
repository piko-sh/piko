package main

type Handler struct {
	cb func(int) int
}

func newHandler(scale int) *Handler {
	return &Handler{
		cb: func(n int) int { return n * scale },
	}
}

func (h *Handler) Call(n int) int {
	return h.cb(n)
}

func run() int {
	h := newHandler(3)
	return h.Call(5) + h.Call(7)
}
