package main

import "fmt"

type Counter struct {
	N int
}

func (c *Counter) Add(delta int) int {
	c.N += delta
	return c.N
}

func (c Counter) Value() int {
	return c.N
}

type Token string

func (t Token) Length() int {
	return len(t)
}

func run() string {
	result := ""

	c := &Counter{N: 10}
	bound := c.Add
	bound(5)
	bound(3)
	result += fmt.Sprintf("bound:c.N=%d;", c.N)

	c2 := Counter{N: 100}
	valueBound := c2.Value
	result += fmt.Sprintf("valueBound:%d;", valueBound())

	expr := (*Counter).Add
	c3 := &Counter{N: 0}
	expr(c3, 7)
	expr(c3, 8)
	result += fmt.Sprintf("expr:c3.N=%d;", c3.N)

	t := Token("hello")
	tBound := t.Length
	result += fmt.Sprintf("tBound:%d;", tBound())

	tExpr := Token.Length
	result += fmt.Sprintf("tExpr:%d", tExpr(Token("worlds")))

	return result
}
