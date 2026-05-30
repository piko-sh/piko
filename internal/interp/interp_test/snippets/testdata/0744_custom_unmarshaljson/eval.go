package main

import (
	"encoding/json"
	"fmt"
)

type Celsius float64

func (c *Celsius) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err == nil {
		switch raw {
		case "freezing":
			*c = 0
			return nil
		case "boiling":
			*c = 100
			return nil
		}
	}
	var n float64
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*c = Celsius(n)
	return nil
}

func run() string {
	result := ""
	tests := []string{`"freezing"`, `"boiling"`, `25.5`, `0`}
	for _, input := range tests {
		var c Celsius
		err := json.Unmarshal([]byte(input), &c)
		result += fmt.Sprintf("%s->%v(err=%v);", input, float64(c), err)
	}
	return result
}
