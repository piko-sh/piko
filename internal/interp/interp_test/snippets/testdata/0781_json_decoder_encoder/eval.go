package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type Record struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func run() string {
	input := `{"id":1,"name":"alice"}
{"id":2,"name":"bob"}
{"id":3,"name":"carol"}
`
	dec := json.NewDecoder(strings.NewReader(input))
	var records []Record
	for dec.More() {
		var r Record
		if err := dec.Decode(&r); err != nil {
			return fmt.Sprintf("decode_err=%v", err)
		}
		records = append(records, r)
	}

	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	for _, r := range records {
		_ = enc.Encode(r)
	}

	result := fmt.Sprintf("count=%d;", len(records))
	for _, r := range records {
		result += fmt.Sprintf("%d/%s,", r.ID, r.Name)
	}
	result += fmt.Sprintf(";out_lines=%d", strings.Count(out.String(), "\n"))
	return result
}
