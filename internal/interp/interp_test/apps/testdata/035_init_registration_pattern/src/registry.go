package main

var handlers = map[string]func() string{}

func register(name string, fn func() string) {
	handlers[name] = fn
}

func dispatch(name string) string {
	if fn, ok := handlers[name]; ok {
		return fn()
	}
	return ""
}
