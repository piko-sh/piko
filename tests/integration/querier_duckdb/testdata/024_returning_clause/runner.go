package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"

	"querier_test_runner/db"
)

func main() {
	conn, err := sql.Open("duckdb", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, `CREATE TABLE items (
		id INTEGER PRIMARY KEY,
		name VARCHAR NOT NULL,
		price INTEGER NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	inserted, err := queries.InsertReturning(ctx, db.InsertReturningParams{P1: int32(1), P2: "Widget", P3: int32(1999)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertReturning:", err)
		os.Exit(1)
	}

	updated, err := queries.UpdateReturning(ctx, db.UpdateReturningParams{P1: int32(1), P2: int32(2499)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "UpdateReturning:", err)
		os.Exit(1)
	}

	deleted, err := queries.DeleteReturning(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "DeleteReturning:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"insert_returning": inserted,
		"update_returning": updated,
		"delete_returning": deleted,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
