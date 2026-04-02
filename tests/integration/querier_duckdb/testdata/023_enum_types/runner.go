package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"

	"querier_test_runner/db"
)

func main() {
	conn, err := sql.Open("duckdb", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, `CREATE TYPE mood AS ENUM ('happy', 'neutral', 'sad')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE TABLE people (
		id INTEGER PRIMARY KEY,
		name VARCHAR NOT NULL,
		current_mood mood NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	if err := queries.InsertPerson(ctx, db.InsertPersonParams{P1: int32(1), P2: "Alice", P3: "happy"}); err != nil {
		fmt.Fprintln(os.Stderr, "InsertPerson 1:", err)
		os.Exit(1)
	}
	if err := queries.InsertPerson(ctx, db.InsertPersonParams{P1: int32(2), P2: "Bob", P3: "sad"}); err != nil {
		fmt.Fprintln(os.Stderr, "InsertPerson 2:", err)
		os.Exit(1)
	}
	if err := queries.InsertPerson(ctx, db.InsertPersonParams{P1: int32(3), P2: "Charlie", P3: "happy"}); err != nil {
		fmt.Fprintln(os.Stderr, "InsertPerson 3:", err)
		os.Exit(1)
	}

	person, err := queries.GetPerson(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetPerson:", err)
		os.Exit(1)
	}

	byMood, err := queries.ListByMood(ctx, "happy")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListByMood:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"get_person":   person,
		"list_by_mood": byMood,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
