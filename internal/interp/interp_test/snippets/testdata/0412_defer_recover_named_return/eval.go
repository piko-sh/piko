package main

func explode() {
	panic("boom")
}

func runSafely() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = r.(string)
		}
	}()
	explode()
	return ""
}

func run() string {
	return runSafely()
}
