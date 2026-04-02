package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "modernc.org/sqlite"

	"querier_test_runner/db"
)

func main() {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, `CREATE TABLE events (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		timestamp INTEGER NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO events (id, name, timestamp) VALUES
		(1, 'login', 1000),
		(2, 'logout', 2000),
		(3, 'login', 3000),
		(4, 'purchase', 4000),
		(5, 'login', 5000)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	var allEvents []db.StreamEventsRow
	for row, err := range queries.StreamEvents(ctx) {
		if err != nil {
			fmt.Fprintln(os.Stderr, "StreamEvents:", err)
			os.Exit(1)
		}
		allEvents = append(allEvents, row)
	}
	var loginEvents []db.StreamEventsByNameRow
	for row, err := range queries.StreamEventsByName(ctx, "login") {
		if err != nil {
			fmt.Fprintln(os.Stderr, "StreamEventsByName:", err)
			os.Exit(1)
		}
		loginEvents = append(loginEvents, row)
	}
	var firstTwo []db.StreamEventsRow
	for row, err := range queries.StreamEvents(ctx) {
		if err != nil {
			fmt.Fprintln(os.Stderr, "StreamEvents (first two):", err)
			os.Exit(1)
		}
		firstTwo = append(firstTwo, row)
		if len(firstTwo) == 2 {
			break
		}
	}

	result := map[string]any{
		"all_events":   allEvents,
		"login_events": loginEvents,
		"first_two":    firstTwo,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
