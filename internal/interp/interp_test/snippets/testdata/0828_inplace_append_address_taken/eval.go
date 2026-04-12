package main

func run() string {
	output := make([]byte, 0, 8)
	output = append(output, 1)
	pointer := &output

	output = append(output, 2)

	if len(*pointer) != 2 {
		return "pointer view stale"
	}
	if (*pointer)[0] != 1 || (*pointer)[1] != 2 {
		return "pointer contents wrong"
	}

	*pointer = append(*pointer, 3)
	if len(output) != 3 {
		return "output view stale"
	}
	if output[0] != 1 || output[1] != 2 || output[2] != 3 {
		return "output contents wrong"
	}
	return "ok"
}
