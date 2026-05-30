package main

func run() string {
	output := make([]byte, 0, 8)
	output = append(output, 1)

	snapshotLen := func() int {
		return len(output)
	}

	if snapshotLen() != 1 {
		return "pre snapshot wrong"
	}

	output = append(output, 2)
	output = append(output, 3)

	if snapshotLen() != 3 {
		return "post snapshot wrong"
	}
	if len(output) != 3 {
		return "outer length wrong"
	}
	if output[0] != 1 || output[1] != 2 || output[2] != 3 {
		return "outer contents wrong"
	}
	return "ok"
}
