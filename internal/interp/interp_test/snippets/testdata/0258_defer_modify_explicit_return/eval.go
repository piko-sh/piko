package main

func modified() (result int) {
	defer func() { result += 10 }()
	return 32
}

func run() int {
	return modified()
}
