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

	_, err = conn.ExecContext(ctx, `INSERT INTO tasks (id, status, priority, title, active) VALUES
		('t1', 'PENDING', 1, 'Low pending', 1),
		('t2', 'PENDING', 3, 'High pending', 1),
		('t3', 'PROCESSING', 2, 'Mid processing', 1),
		('t4', 'COMPLETE', 1, 'Done task', 1),
		('t5', 'FAILED', 3, 'High failed', 1),
		('t6', 'RETRYING', 2, 'Mid retrying', 1)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	fetched, err := queries.FetchByPriorityStatusesAndActive(ctx, db.FetchByPriorityStatusesAndActiveParams{
		P1:       int32(2),
		Statuses: []string{"PENDING", "PROCESSING", "RETRYING"},
		P3:       int32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByPriorityStatusesAndActive:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"fetch_by_priority_statuses_and_active": fetched,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
