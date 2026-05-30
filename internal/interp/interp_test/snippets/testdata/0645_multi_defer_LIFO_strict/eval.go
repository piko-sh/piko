package main

func run() string {
	out := ""
	defer func() { out += "1" }()
	defer func() { out += "2" }()
	defer func() { out += "3" }()
	_ = out
	return ""
}
