package main

func sumInts() int64 {
	out := []int64{}
	for index := int64(0); index < 8; index++ {
		out = append(out, index)
	}
	more := []int64{100, 200, 300}
	out = append(out, more...)

	var total int64
	for _, value := range out {
		total += value
	}
	return total
}

func sumUints() uint64 {
	out := []uint64{}
	for index := uint64(0); index < 8; index++ {
		out = append(out, index)
	}
	var total uint64
	for _, value := range out {
		total += value
	}
	return total
}

func sumFloats() float64 {
	out := []float64{}
	for index := 0; index < 8; index++ {
		out = append(out, float64(index)+0.5)
	}
	var total float64
	for _, value := range out {
		total += value
	}
	return total
}

func anyTrue() bool {
	out := []bool{}
	for index := 0; index < 5; index++ {
		out = append(out, index%2 == 1)
	}
	for _, value := range out {
		if value {
			return true
		}
	}
	return false
}

func concatStrings() string {
	out := []string{}
	for index := 0; index < 3; index++ {
		out = append(out, "x")
	}
	more := []string{"y", "z"}
	out = append(out, more...)

	result := ""
	for _, value := range out {
		result += value
	}
	return result
}

func run() string {
	intsTotal := sumInts()
	uintsTotal := sumUints()
	floatsTotal := sumFloats()
	flag := anyTrue()
	tail := concatStrings()

	result := ""
	if intsTotal == 628 {
		result += "I"
	}
	if uintsTotal == 28 {
		result += "U"
	}
	if floatsTotal > 30.0 && floatsTotal < 35.0 {
		result += "F"
	}
	if flag {
		result += "B"
	}
	result += "-"
	result += tail
	return result
}
