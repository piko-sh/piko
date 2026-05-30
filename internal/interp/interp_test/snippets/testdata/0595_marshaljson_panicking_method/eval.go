package main

import "encoding/json"

type Bomb struct{}

func (b *Bomb) MarshalJSON() ([]byte, error) {
	panic("marshal boom")
}

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	_, _ = json.Marshal(&Bomb{})
	return 99
}
