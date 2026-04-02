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

	_, err = conn.ExecContext(ctx, `CREATE TABLE source (id INTEGER PRIMARY KEY, name TEXT NOT NULL, value REAL)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO source (id, name, value) VALUES (1, 'alpha', 10.5), (2, 'beta', NULL), (3, 'gamma', 30.0)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE TABLE derived AS SELECT id, name FROM source WHERE value > 0`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	sourceRows, err := queries.ListSource(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	derivedRows, err := conn.QueryContext(ctx, `SELECT id, name FROM derived ORDER BY id`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer derivedRows.Close()

	type derivedRow struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	var derived []derivedRow
	for derivedRows.Next() {
		var row derivedRow
		if err := derivedRows.Scan(&row.ID, &row.Name); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		derived = append(derived, row)
	}
	if err := derivedRows.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := map[string]any{
		"source":  sourceRows,
		"derived": derived,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
