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
		('t1', 'PENDING', 1, 'Task 1'),
		('t2', 'PENDING', 2, 'Task 2'),
		('t3', 'PROCESSING', 3, 'Task 3'),
		('t4', 'COMPLETE', 1, 'Task 4'),
		('t5', 'FAILED', 2, 'Task 5'),
		('t6', 'RETRYING', 3, 'Task 6')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	deleted, err := queries.DeleteByStatusAndIDs(ctx, db.DeleteByStatusAndIDsParams{
		P1:  "PENDING",
		IDs: []string{"t1", "t2"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "DeleteByStatusAndIDs:", err)
		os.Exit(1)
	}
	err = queries.UpdateStatusByIDs(ctx, db.UpdateStatusByIDsParams{
		P1:  "ARCHIVED",
		IDs: []string{"t3", "t6"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "UpdateStatusByIDs:", err)
		os.Exit(1)
	}
	countRow, err := queries.CountNonArchived(ctx, "ARCHIVED")
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountNonArchived:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"deleted":          deleted,
		"remaining_active": countRow.Total,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
