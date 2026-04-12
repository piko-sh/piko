package main

func run() string {
	scores := make([]float64, 3)
	scores[0] = 95.5
	scores[1] = 87.0
	scores[2] = 72.5
	labels := make([]string, 3)
	labels[0] = "first"
	labels[1] = "second"
	labels[2] = "third"
	flags := make([]bool, 3)
	flags[0] = true
	flags[1] = false
	flags[2] = true
	out := ""
	for i := 0; i < len(scores); i++ {
		if flags[i] && scores[i] >= 80.0 {
			if out != "" {
				out += ","
			}
			out += labels[i]
		}
	}
	return out
}
