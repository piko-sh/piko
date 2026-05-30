package main

import (
	"fmt"
	"time"
)

type Event struct {
	time.Time
	Label string
}

func run() string {
	t := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	event := Event{Time: t, Label: "launch"}

	added := event.Add(2 * time.Hour)
	diff := added.Sub(event.Time)
	formatted := event.Format("2006-01-02 15:04")
	return fmt.Sprintf("label=%s;fmt=%s;diff=%v;year=%d",
		event.Label, formatted, diff, event.Year())
}
