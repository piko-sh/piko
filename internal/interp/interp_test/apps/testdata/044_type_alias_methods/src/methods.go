package main

func (s Series) Len() int {
	return len(s.values)
}

func (s Series) Sum() int {
	total := 0
	for _, v := range s.values {
		total += v
	}
	return total
}

func (s Series) First() int {
	if len(s.values) == 0 {
		return 0
	}
	return s.values[0]
}
