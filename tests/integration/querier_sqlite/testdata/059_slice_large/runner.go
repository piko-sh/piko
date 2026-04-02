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

	_, err = conn.ExecContext(ctx, `CREATE TABLE items (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for i := int32(1); i <= 200; i++ {
		_, err = conn.ExecContext(ctx, `INSERT INTO items (id, name) VALUES (?, ?)`, i, fmt.Sprintf("item_%d", i))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	queries := db.New(conn)

	ids := make([]int32, 150)
	for i := range ids {
		ids[i] = int32(i + 1)
	}

	rows, err := queries.FetchByIDs(ctx, db.FetchByIDsParams{
		IDs: ids,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByIDs:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"count": len(rows),
		"first": rows[0],
		"last":  rows[len(rows)-1],
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
