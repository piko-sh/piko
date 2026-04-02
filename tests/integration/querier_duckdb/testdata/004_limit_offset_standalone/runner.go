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
		name TEXT NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO items (id, name) VALUES
		(1, 'item_1'), (2, 'item_2'), (3, 'item_3'), (4, 'item_4'), (5, 'item_5'),
		(6, 'item_6'), (7, 'item_7'), (8, 'item_8'), (9, 'item_9'), (10, 'item_10')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	defaultPage, err := queries.ListItems(ctx, db.ListItemsParams{
		PageSize:   0,
		PageOffset: 0,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListItems (default):", err)
		os.Exit(1)
	}
	firstFive, err := queries.ListItems(ctx, db.ListItemsParams{
		PageSize:   5,
		PageOffset: 0,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListItems (first five):", err)
		os.Exit(1)
	}
	lastFive, err := queries.ListItems(ctx, db.ListItemsParams{
		PageSize:   5,
		PageOffset: 5,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListItems (last five):", err)
		os.Exit(1)
	}
	clampedToMax, err := queries.ListItems(ctx, db.ListItemsParams{
		PageSize:   20,
		PageOffset: 0,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListItems (clamped):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"default_page":   defaultPage,
		"first_five":     firstFive,
		"last_five":      lastFive,
		"clamped_to_max": clampedToMax,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
