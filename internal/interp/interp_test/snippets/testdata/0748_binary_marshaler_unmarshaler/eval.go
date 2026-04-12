package main

import (
	"encoding"
	"encoding/binary"
	"fmt"
)

type Point struct {
	X, Y int32
}

func (p Point) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(p.X))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(p.Y))
	return buf, nil
}

func (p *Point) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("short")
	}
	p.X = int32(binary.LittleEndian.Uint32(data[0:4]))
	p.Y = int32(binary.LittleEndian.Uint32(data[4:8]))
	return nil
}

func run() string {
	result := ""
	original := Point{X: 1000, Y: -42}
	var marshaller encoding.BinaryMarshaler = original
	encoded, _ := marshaller.MarshalBinary()
	result += fmt.Sprintf("len=%d;", len(encoded))

	var decoded Point
	var unmarshaller encoding.BinaryUnmarshaler = &decoded
	err := unmarshaller.UnmarshalBinary(encoded)
	result += fmt.Sprintf("decoded=%d/%d,err=%v", decoded.X, decoded.Y, err)
	return result
}
