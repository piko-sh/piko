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
	fetched, err := queries.FetchByPriorityAndStatuses(ctx, db.FetchByPriorityAndStatusesParams{
		P1:       int32(2),
		Statuses: []string{"PENDING", "RETRYING"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByPriorityAndStatuses:", err)
		os.Exit(1)
	}
	counted, err := queries.CountByStatuses(ctx, db.CountByStatusesParams{
		Statuses: []string{"PENDING", "PROCESSING"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountByStatuses:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"fetch_by_priority_and_statuses": fetched,
		"count_by_statuses":             counted,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
