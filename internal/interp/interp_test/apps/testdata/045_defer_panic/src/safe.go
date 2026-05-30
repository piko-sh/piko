package main

func runSafely() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = r.(string)
		}
	}()
	explode()
	return ""
}
