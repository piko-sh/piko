package main

type IntList []int

func (l IntList) Sum() int {
	total := 0
	for _, v := range l {
		total += v
	}
	return total
}

func (l IntList) Max() int {
	if len(l) == 0 {
		return 0
	}
	m := l[0]
	for _, v := range l[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func run() int {
	l := IntList{3, 1, 4, 1, 5, 9, 2, 6}
	return l.Sum() + l.Max()
}
