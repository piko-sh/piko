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

	if _, err = conn.ExecContext(ctx, `CREATE TABLE docs (id INTEGER PRIMARY KEY, payload JSON)`); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if _, err = conn.ExecContext(ctx, `INSERT INTO docs (id, payload) VALUES (1, NULL), (2, '{"k":"v"}')`); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	nullRow, err := queries.GetDoc(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetDoc(1) NULL payload:", err)
		os.Exit(1)
	}

	populatedRow, err := queries.GetDoc(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetDoc(2) populated payload:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"null_payload":      nullRow,
		"populated_payload": populatedRow,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
