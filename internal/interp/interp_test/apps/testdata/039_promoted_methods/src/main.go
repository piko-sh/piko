package main

import "fmt"

func entrypoint() string {
	s := ServiceDog{Dog: Dog{Animal: Animal{Name: "Rex"}}, Skill: "guide"}
	return fmt.Sprintf("name=%s kind=%s skill=%s breathe=%s", s.Name, s.kind(), s.skill(), s.breathe())
}

func main() {
	fmt.Println(entrypoint())
}
