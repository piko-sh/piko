package main

func lookup(domain string, path string) (int, bool) {
	weight, ok := registry[struct {
		Domain string
		Path   string
	}{Domain: domain, Path: path}]
	return weight, ok
}
