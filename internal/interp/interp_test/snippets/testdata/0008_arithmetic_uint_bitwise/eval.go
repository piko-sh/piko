package main

var a uint = 0xFF
var b uint = 0x0F

func run() int {
	return int(a&b) + int(a|b) + int(a^b) + int(a&^b)
}
