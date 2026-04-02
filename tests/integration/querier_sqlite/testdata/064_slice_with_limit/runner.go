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
		title TEXT NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO tasks (id, status, priority, title) VALUES
		('t1', 'PENDING', 1, 'Low pending'),
		('t2', 'PENDING', 3, 'High pending'),
		('t3', 'PROCESSING', 2, 'Mid processing'),
		('t4', 'COMPLETE', 1, 'Done task'),
		('t5', 'FAILED', 3, 'High failed'),
		('t6', 'RETRYING', 2, 'Mid retrying')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	fetchLimited, err := queries.FetchByStatusesLimited(ctx, db.FetchByStatusesLimitedParams{
		Statuses: []string{"PENDING", "PROCESSING", "RETRYING"},
		PageSize: int(2),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByStatusesLimited:", err)
		os.Exit(1)
	}
	countResult, err := queries.CountByStatuses(ctx, db.CountByStatusesParams{
		Statuses: []string{"PENDING", "PROCESSING", "RETRYING"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountByStatuses:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"fetch_limited": fetchLimited,
		"count":         countResult,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
