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

	_, err = conn.ExecContext(ctx, `CREATE TABLE counters (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		value INTEGER NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO counters (id, name, value) VALUES
		(1, 'hits', 0),
		(2, 'errors', 0)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	hitsResult, err := queries.IncrementCounter(ctx, db.IncrementCounterParams{
		P1: 5,
		P2: "hits",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "IncrementCounter (hits):", err)
		os.Exit(1)
	}
	hitsAffected, err := hitsResult.RowsAffected()
	if err != nil {
		fmt.Fprintln(os.Stderr, "RowsAffected (hits):", err)
		os.Exit(1)
	}
	errorsResult, err := queries.IncrementCounter(ctx, db.IncrementCounterParams{
		P1: 1,
		P2: "errors",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "IncrementCounter (errors):", err)
		os.Exit(1)
	}
	errorsAffected, err := errorsResult.RowsAffected()
	if err != nil {
		fmt.Fprintln(os.Stderr, "RowsAffected (errors):", err)
		os.Exit(1)
	}
	missingResult, err := queries.IncrementCounter(ctx, db.IncrementCounterParams{
		P1: 1,
		P2: "missing",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "IncrementCounter (missing):", err)
		os.Exit(1)
	}
	missingAffected, err := missingResult.RowsAffected()
	if err != nil {
		fmt.Fprintln(os.Stderr, "RowsAffected (missing):", err)
		os.Exit(1)
	}
	hitsCounter, err := queries.GetCounter(ctx, "hits")
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetCounter (hits):", err)
		os.Exit(1)
	}

	errorsCounter, err := queries.GetCounter(ctx, "errors")
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetCounter (errors):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"hits_affected":   hitsAffected,
		"hits_value":      hitsCounter.Value,
		"errors_affected": errorsAffected,
		"errors_value":    errorsCounter.Value,
		"missing_affected": missingAffected,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
