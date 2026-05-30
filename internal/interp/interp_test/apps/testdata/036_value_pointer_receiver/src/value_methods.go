package main

func (c counter) describe() string {
	return c.label
}

func (c counter) doubled() int {
	return c.value * 2
}
