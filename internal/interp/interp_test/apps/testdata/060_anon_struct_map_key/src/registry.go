package main

var registry = map[struct {
	Domain string
	Path   string
}]int{}

func register(domain string, path string, weight int) {
	registry[struct {
		Domain string
		Path   string
	}{Domain: domain, Path: path}] = weight
}
