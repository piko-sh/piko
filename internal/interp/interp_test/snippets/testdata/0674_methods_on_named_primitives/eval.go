package main

import "fmt"

type Status int

const (
	StatusUnknown Status = iota
	StatusPending
	StatusActive
	StatusClosed
)

func (s Status) String() string {
	switch s {
	case StatusUnknown:
		return "unknown"
	case StatusPending:
		return "pending"
	case StatusActive:
		return "active"
	case StatusClosed:
		return "closed"
	}
	return "?"
}

func (s Status) IsTerminal() bool {
	return s == StatusClosed
}

type Tag string

func (t Tag) Quoted() string {
	return "[" + string(t) + "]"
}

func (t Tag) Length() int {
	return len(t)
}

type Counter uint32

func (c Counter) Next() Counter {
	return c + 1
}

func (c Counter) Format() string {
	return fmt.Sprintf("c=%d", c)
}

func run() string {
	result := ""

	statuses := []Status{StatusUnknown, StatusPending, StatusActive, StatusClosed}
	for _, s := range statuses {
		result += s.String() + "/" + boolToStr(s.IsTerminal()) + ";"
	}

	tag := Tag("hello")
	result += tag.Quoted() + "/" + intToStr(int64(tag.Length())) + ";"

	var c Counter = 7
	c = c.Next().Next()
	result += c.Format()

	return result
}

func boolToStr(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func intToStr(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
