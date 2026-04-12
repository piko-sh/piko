package main

func (e employee) jobTag() string {
	return e.role + "-" + e.name
}
