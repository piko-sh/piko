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

	_, err = conn.ExecContext(ctx, `CREATE TABLE kv (
		key VARCHAR PRIMARY KEY,
		value VARCHAR NOT NULL,
		version INTEGER NOT NULL DEFAULT 1
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	if err := queries.Upsert(ctx, db.UpsertParams{P1: "foo", P2: "bar"}); err != nil {
		fmt.Fprintln(os.Stderr, "Upsert 1:", err)
		os.Exit(1)
	}

	afterInsert, err := queries.Get(ctx, "foo")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Get after insert:", err)
		os.Exit(1)
	}
	if err := queries.Upsert(ctx, db.UpsertParams{P1: "foo", P2: "baz"}); err != nil {
		fmt.Fprintln(os.Stderr, "Upsert 2:", err)
		os.Exit(1)
	}

	afterUpsert, err := queries.Get(ctx, "foo")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Get after upsert:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"after_insert": afterInsert,
		"after_upsert": afterUpsert,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
