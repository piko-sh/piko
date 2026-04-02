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
	sortedByPriorityDesc, err := queries.FetchByStatusesSorted(ctx, db.FetchByStatusesSortedParams{
		Statuses:         []string{"PENDING", "RETRYING"},
		OrderBy:          db.FetchByStatusesSortedOrderByPriority,
		OrderByDirection: db.OrderDesc,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByStatusesSorted (priority DESC):", err)
		os.Exit(1)
	}
	sortedByIDAsc, err := queries.FetchByStatusesSorted(ctx, db.FetchByStatusesSortedParams{
		Statuses:         []string{"PENDING", "PROCESSING", "RETRYING"},
		OrderBy:          db.FetchByStatusesSortedOrderByID,
		OrderByDirection: db.OrderAsc,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByStatusesSorted (id ASC):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"sorted_by_priority_desc": sortedByPriorityDesc,
		"sorted_by_id_asc":       sortedByIDAsc,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
