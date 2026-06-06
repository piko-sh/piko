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

	_, err = conn.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO users (id, name) VALUES (1, 'Alice'), (2, 'Bob'), (3, 'Carol')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	byIDs, err := queries.GetUsersByIDs(ctx, db.GetUsersByIDsParams{IDs: []int32{1, 3}})
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetUsersByIDs:", err)
		os.Exit(1)
	}

	byEmpty, err := queries.GetUsersByIDs(ctx, db.GetUsersByIDsParams{IDs: []int32{}})
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetUsersByIDs (empty):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"by_ids":   byIDs,
		"by_empty": byEmpty,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
