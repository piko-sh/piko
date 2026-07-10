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
	conn, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()
	if _, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", os.Getenv("DATABASE_SCHEMA"))); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	uniqueKey := "job-unique-key"
	correlationID := "corr-1"
	row, err := queries.InsertJobRootWithUniqueKey(ctx, db.InsertJobRootWithUniqueKeyParams{
		ID:             "job-1",
		Kind:           "email",
		Queue:          "default",
		Payload:        "{}",
		UniqueKey:      &uniqueKey,
		CorrelationID:  &correlationID,
		MaxAttempts:    3,
		TimeoutSeconds: 30,
		CreatedAt:      "2026-07-10T00:00:00Z",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(map[string]any{"inserted_id": row.ID}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
