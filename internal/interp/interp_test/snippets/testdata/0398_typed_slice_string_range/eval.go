package main

func run() string {
	s := make([]string, 4)
	s[0] = "alpha"
	s[1] = "beta"
	s[2] = "gamma"
	s[3] = "delta"
	out := ""
	for i, v := range s {
		if i > 0 {
			out += "-"
		}
		out += v
	}
	return out
}
