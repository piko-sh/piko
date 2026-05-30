package main

func run() string {
	output := make([]byte, 0, 16)
	output = append(output, 1)
	saved := output
	output = append(output, 2)
	output = append(output, 3)

	if len(saved) != 1 {
		return "saved length changed"
	}
	if saved[0] != 1 {
		return "saved[0] changed"
	}
	if len(output) != 3 {
		return "output length wrong"
	}
	if output[0] != 1 || output[1] != 2 || output[2] != 3 {
		return "output contents wrong"
	}
	return "ok"
}
