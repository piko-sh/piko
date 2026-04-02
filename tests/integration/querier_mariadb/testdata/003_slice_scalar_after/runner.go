package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"querier_test_runner/db"
)

func main() {
	connectionString := os.Getenv("DATABASE_URL")

	conn, err := sql.Open("mysql", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

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
	fetched, err := queries.FetchByStatusesAndPriority(ctx, db.FetchByStatusesAndPriorityParams{
		Statuses: []string{"PENDING", "RETRYING"},
		P2:       int32(2),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByStatusesAndPriority:", err)
		os.Exit(1)
	}
	counted, err := queries.CountByStatusesAndPriority(ctx, db.CountByStatusesAndPriorityParams{
		Statuses: []string{"PENDING", "PROCESSING", "RETRYING"},
		P2:       int32(2),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountByStatusesAndPriority:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"fetch_by_statuses_and_priority": fetched,
		"count_by_statuses_and_priority": counted,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
