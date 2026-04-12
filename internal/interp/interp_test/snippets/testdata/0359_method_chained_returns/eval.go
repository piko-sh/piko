package main

type Builder struct {
	parts []string
}

func (b *Builder) Add(s string) *Builder {
	b.parts = append(b.parts, s)
	return b
}

func (b *Builder) Result() string {
	out := ""
	for _, p := range b.parts {
		out += p
	}
	return out
}

func run() string {
	return (&Builder{}).Add("a").Add("b").Add("c").Result()
}
