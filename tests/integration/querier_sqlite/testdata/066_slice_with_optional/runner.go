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

	withoutFilter, err := queries.FetchByStatusesWithOptionalPriority(ctx, db.FetchByStatusesWithOptionalPriorityParams{
		Statuses: []string{"PENDING", "RETRYING"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByStatusesWithOptionalPriority (no filter):", err)
		os.Exit(1)
	}

	minPriority2 := int32(2)
	withMinPriority2, err := queries.FetchByStatusesWithOptionalPriority(ctx, db.FetchByStatusesWithOptionalPriorityParams{
		Statuses:    []string{"PENDING", "PROCESSING", "RETRYING"},
		MinPriority: &minPriority2,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByStatusesWithOptionalPriority (min_priority=2):", err)
		os.Exit(1)
	}

	minPriority3 := int32(3)
	withMinPriority3, err := queries.FetchByStatusesWithOptionalPriority(ctx, db.FetchByStatusesWithOptionalPriorityParams{
		Statuses:    []string{"PENDING", "RETRYING"},
		MinPriority: &minPriority3,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByStatusesWithOptionalPriority (min_priority=3):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"without_filter":    withoutFilter,
		"with_min_priority_2": withMinPriority2,
		"with_min_priority_3": withMinPriority3,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
