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
	updatedRows, err := queries.UpdateStatusByIDs(ctx, db.UpdateStatusByIDsParams{
		P1:  "UPDATED",
		IDs: []string{"t1", "t3"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "UpdateStatusByIDs:", err)
		os.Exit(1)
	}
	fetched, err := queries.FetchByPriorityAndStatuses(ctx, db.FetchByPriorityAndStatusesParams{
		P1:       int32(2),
		Statuses: []string{"PROCESSING", "RETRYING", "UPDATED"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByPriorityAndStatuses:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"updated_rows":                updatedRows,
		"fetch_by_priority_and_statuses": fetched,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
