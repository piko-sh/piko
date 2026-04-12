package main

import (
	"fmt"
)

type Money struct {
	Amount int64
	Code   string
}

func (m Money) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "Money{Amount: %d, Code: %s}", m.Amount, m.Code)
		} else {
			fmt.Fprintf(s, "%s %d", m.Code, m.Amount)
		}
	case 's':
		fmt.Fprintf(s, "%s%d", m.Code, m.Amount)
	default:
		fmt.Fprintf(s, "!"+string(verb)+"(Money)")
	}
}

func run() string {
	m := Money{Amount: 999, Code: "GBP"}
	result := ""
	result += fmt.Sprintf("v=%v;", m)
	result += fmt.Sprintf("plus=%+v;", m)
	result += fmt.Sprintf("s=%s;", m)
	result += fmt.Sprintf("d=%d", m)
	return result
}
