package main

type Info struct {
	Title string
}

func lookup(m map[string]*Info, k string) string {
	if info, ok := m[k]; ok {
		return info.Title
	}
	return "none"
}

func run() string {
	m := map[string]*Info{"x": {Title: "found"}}
	return lookup(m, "x") + "-" + lookup(m, "y")
}
