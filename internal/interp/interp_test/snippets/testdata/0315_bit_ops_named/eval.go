package main

type Flags uint32

const (
	FlagA Flags = 1 << iota
	FlagB
	FlagC
	FlagD
)

func run() int {
	mask := FlagA | FlagC
	if mask&FlagA != 0 && mask&FlagB == 0 && mask&FlagC != 0 {
		return int(mask)
	}
	return -1
}
