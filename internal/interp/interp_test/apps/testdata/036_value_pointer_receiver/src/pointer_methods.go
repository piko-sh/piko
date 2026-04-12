package main

func (c *counter) bump() {
	c.value++
}

func (c *counter) reset() {
	c.value = 0
}
