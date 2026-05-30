package main

type ServiceDog struct {
	Dog
	Skill string
}

func (s ServiceDog) skill() string {
	return s.Skill
}
