package main

func run() int {
	s := "héllo"
	runeCount := 0
	totalCode := 0
	for _, r := range s {
		runeCount++
		totalCode += int(r)
	}
	return runeCount*10000 + totalCode
}
