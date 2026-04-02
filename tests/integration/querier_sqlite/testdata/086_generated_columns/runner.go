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
		price INTEGER NOT NULL,
		quantity INTEGER NOT NULL,
		total INTEGER GENERATED ALWAYS AS (price * quantity) STORED,
		label TEXT GENERATED ALWAYS AS (printf('Item #%d', id)) STORED
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	err = queries.InsertItem(ctx, db.InsertItemParams{
		P1: int32(1),
		P2: int32(10),
		P3: int32(5),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertItem 1:", err)
		os.Exit(1)
	}

	err = queries.InsertItem(ctx, db.InsertItemParams{
		P1: int32(2),
		P2: int32(25),
		P3: int32(2),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertItem 2:", err)
		os.Exit(1)
	}

	err = queries.InsertItem(ctx, db.InsertItemParams{
		P1: int32(3),
		P2: int32(100),
		P3: int32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertItem 3:", err)
		os.Exit(1)
	}

	item, err := queries.GetItem(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetItem:", err)
		os.Exit(1)
	}

	minTotal := int32(50)
	byMinTotal, err := queries.ListByMinTotal(ctx, &minTotal)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListByMinTotal:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"item":         item,
		"by_min_total": byMinTotal,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
