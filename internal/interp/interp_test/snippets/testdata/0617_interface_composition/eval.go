package main

type Stringer interface {
	String() string
}

type Tagger interface {
	Tag() string
}

type Both interface {
	Stringer
	Tagger
}

type item struct{}

func (item) String() string { return "s" }
func (item) Tag() string    { return "t" }

func run() string {
	var b Both = item{}
	return b.String() + "|" + b.Tag()
}
