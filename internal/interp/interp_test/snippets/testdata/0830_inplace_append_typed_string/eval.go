package main

func run() string {
	words := make([]string, 0, 4)
	words = append(words, "the")
	words = append(words, "quick")
	words = append(words, "brown")
	words = append(words, "fox")
	if len(words) != 4 {
		return "wrong length"
	}
	if words[0] != "the" || words[1] != "quick" || words[2] != "brown" || words[3] != "fox" {
		return "wrong contents"
	}
	words = append(words, "jumps")
	if len(words) != 5 || words[4] != "jumps" {
		return "grow wrong"
	}
	combined := ""
	for index := 0; index < len(words); index++ {
		if index > 0 {
			combined += " "
		}
		combined += words[index]
	}
	return combined
}
