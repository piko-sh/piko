package main

import (
	"encoding"
	"fmt"
)

type Colour struct {
	R, G, B uint8
}

func (c Colour) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)), nil
}

func (c *Colour) UnmarshalText(text []byte) error {
	var r, g, b uint8
	_, err := fmt.Sscanf(string(text), "#%02X%02X%02X", &r, &g, &b)
	if err != nil {
		return err
	}
	c.R, c.G, c.B = r, g, b
	return nil
}

func run() string {
	result := ""
	original := Colour{R: 255, G: 128, B: 64}
	var marshaller encoding.TextMarshaler = original
	encoded, _ := marshaller.MarshalText()
	result += fmt.Sprintf("encoded=%s;", string(encoded))

	var decoded Colour
	var unmarshaller encoding.TextUnmarshaler = &decoded
	err := unmarshaller.UnmarshalText(encoded)
	result += fmt.Sprintf("decoded=%d/%d/%d,err=%v", decoded.R, decoded.G, decoded.B, err)
	return result
}
