package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
)

type Row struct {
	ID   int
	Name string
}

func run() string {
	rows := []Row{{1, "alice"}, {2, "bob"}, {3, "carol"}}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	for _, r := range rows {
		_ = w.Write([]string{strconv.Itoa(r.ID), r.Name})
	}
	w.Flush()

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		return fmt.Sprintf("err=%v", err)
	}

	result := fmt.Sprintf("count=%d;", len(records))
	for _, rec := range records {
		result += fmt.Sprintf("%s/%s,", rec[0], rec[1])
	}
	return result
}
