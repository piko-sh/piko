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

	_, err = conn.ExecContext(ctx, `CREATE TABLE orders (
		id INTEGER PRIMARY KEY,
		customer TEXT NOT NULL,
		total INTEGER NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE TABLE order_items (
		id INTEGER PRIMARY KEY,
		order_id INTEGER NOT NULL REFERENCES orders(id),
		product TEXT NOT NULL,
		quantity INTEGER NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO orders (id, customer, total) VALUES
		(1, 'Alice', 100),
		(2, 'Bob', 200),
		(3, 'Charlie', 0)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO order_items (id, order_id, product, quantity) VALUES
		(1, 1, 'Widget', 2),
		(2, 1, 'Gadget', 1),
		(3, 2, 'Widget', 5)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	ordersWithItems, err := queries.ListOrdersWithItems(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListOrdersWithItems:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"orders": ordersWithItems,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
