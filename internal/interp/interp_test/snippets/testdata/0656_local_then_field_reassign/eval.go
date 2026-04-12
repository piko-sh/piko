package main

type holder struct {
	keys []string
}

func mutate(h *holder) []string {
	oldKeys := h.keys
	h.keys = make([]string, 8)
	for index := 0; index < 8; index++ {
		h.keys[index] = "new"
	}
	return oldKeys
}

func run() string {
	h := &holder{keys: []string{"a", "b", "c", "d"}}
	captured := mutate(h)
	result := ""
	for index := 0; index < len(captured); index++ {
		result += captured[index]
	}
	return result
}
