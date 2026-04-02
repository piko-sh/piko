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

	_, err = conn.ExecContext(ctx, `CREATE TABLE tasks (
		id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		priority INTEGER NOT NULL,
		title TEXT NOT NULL,
		active INTEGER NOT NULL DEFAULT 1
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO tasks (id, status, priority, title) VALUES
		('t1', 'PENDING', 1, 'Low'),
		('t2', 'PENDING', 3, 'High'),
		('t3', 'PROCESSING', 2, 'Mid'),
		('t4', 'COMPLETE', 1, 'Done'),
		('t5', 'FAILED', 3, 'Fail'),
		('t6', 'RETRYING', 2, 'Retry')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	singleResult, err := queries.FetchByIDsAndStatus(ctx, db.FetchByIDsAndStatusParams{
		IDs: []string{"t1"},
		P2:  "PENDING",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByIDsAndStatus (single):", err)
		os.Exit(1)
	}
	multiResult, err := queries.FetchByIDsAndStatus(ctx, db.FetchByIDsAndStatusParams{
		IDs: []string{"t1", "t2", "t3"},
		P2:  "PENDING",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByIDsAndStatus (multi):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"single_element": singleResult,
		"multi_element":  multiResult,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
