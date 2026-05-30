package main

import (
	"encoding/xml"
	"fmt"
)

type Item struct {
	XMLName xml.Name `xml:"item"`
	ID      int      `xml:"id,attr"`
	Name    string   `xml:"name"`
	Price   float64  `xml:"price"`
}

func run() string {
	original := Item{ID: 42, Name: "widget", Price: 9.99}
	data, err := xml.Marshal(original)
	if err != nil {
		return fmt.Sprintf("marshal_err=%v", err)
	}

	var decoded Item
	if err := xml.Unmarshal(data, &decoded); err != nil {
		return fmt.Sprintf("unmarshal_err=%v", err)
	}

	return fmt.Sprintf("id=%d,name=%s,price=%g;xml_len=%d",
		decoded.ID, decoded.Name, decoded.Price, len(data))
}
