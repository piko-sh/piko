package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"querier_test_runner/db"
)

func main() {
	connectionString := os.Getenv("DATABASE_URL")
	schemaName := os.Getenv("DATABASE_SCHEMA")

	conn, err := sql.Open("pgx", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schemaName))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO events (name, timestamp) VALUES
		('login', 1000),
		('logout', 2000),
		('login', 3000),
		('purchase', 4000),
		('login', 5000)`)
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
