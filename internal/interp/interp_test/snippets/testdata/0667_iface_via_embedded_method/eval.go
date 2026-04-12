package main

type Speaker interface {
	Speak() string
}

type Animal struct {
	Name string
}

func (a Animal) Speak() string { return a.Name + "-speaks" }

type Inner struct {
	Tag string
}

func (i *Inner) Tagged() string { return "[" + i.Tag + "]" }

type Dog struct {
	Animal
}

type Cat struct {
	*Animal
}

type Hybrid struct {
	Inner
}

type Outer struct {
	Hybrid
}

func speak(s Speaker) string {
	return s.Speak()
}

func run() string {
	result := ""

	d := Dog{Animal: Animal{Name: "Rex"}}
	result += speak(d) + ";"

	a := &Animal{Name: "Mittens"}
	c := Cat{Animal: a}
	result += speak(c) + ";"

	o := &Outer{Hybrid: Hybrid{Inner: Inner{Tag: "deep"}}}
	result += o.Tagged() + ";"

	return result
}
