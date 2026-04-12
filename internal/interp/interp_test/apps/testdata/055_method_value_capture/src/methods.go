package main

func (g Greeter) Greet(name string) string {
	return g.Prefix + "-" + name
}
