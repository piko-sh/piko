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
	byStatuses, err := queries.FetchByStatuses(ctx, db.FetchByStatusesParams{
		Statuses: []string{"PENDING", "RETRYING"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByStatuses:", err)
		os.Exit(1)
	}
	byStatusAndPriority, err := queries.FetchByStatusesAndPriority(ctx, db.FetchByStatusesAndPriorityParams{
		Statuses: []string{"PENDING", "PROCESSING", "RETRYING"},
		P2:       int32(2),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByStatusesAndPriority:", err)
		os.Exit(1)
	}
	countOne, err := queries.CountByStatuses(ctx, db.CountByStatusesParams{
		Statuses: []string{"COMPLETE"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountByStatuses (one):", err)
		os.Exit(1)
	}
	countMany, err := queries.CountByStatuses(ctx, db.CountByStatusesParams{
		Statuses: []string{"PENDING", "PROCESSING", "RETRYING", "FAILED"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountByStatuses (many):", err)
		os.Exit(1)
	}
	deleted, err := queries.DeleteByIDs(ctx, db.DeleteByIDsParams{
		IDs: []string{"t4", "t5"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "DeleteByIDs:", err)
		os.Exit(1)
	}
	remaining, err := queries.CountByStatuses(ctx, db.CountByStatusesParams{
		Statuses: []string{"PENDING", "PROCESSING", "RETRYING", "FAILED", "COMPLETE"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountByStatuses (remaining):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"fetch_by_statuses":              byStatuses,
		"fetch_by_statuses_and_priority": byStatusAndPriority,
		"count_one_status":               countOne,
		"count_many_statuses":            countMany,
		"deleted_count":                  deleted,
		"remaining_after_delete":         remaining,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
