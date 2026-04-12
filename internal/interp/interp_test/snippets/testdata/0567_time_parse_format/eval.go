package main

import "time"

func run() string {
	t, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
	return t.Format("2006-01-02")
}
