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

	_, err = conn.ExecContext(ctx, `CREATE SEQUENCE order_seq START WITH 1000 INCREMENT BY 1`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE TABLE orders (
		id INTEGER DEFAULT nextval('order_seq') PRIMARY KEY,
		customer VARCHAR NOT NULL,
		total INTEGER NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	if err := queries.InsertOrder(ctx, db.InsertOrderParams{P1: "Alice", P2: int32(5000)}); err != nil {
		fmt.Fprintln(os.Stderr, "InsertOrder 1:", err)
		os.Exit(1)
	}
	if err := queries.InsertOrder(ctx, db.InsertOrderParams{P1: "Bob", P2: int32(3000)}); err != nil {
		fmt.Fprintln(os.Stderr, "InsertOrder 2:", err)
		os.Exit(1)
	}
	if err := queries.InsertOrder(ctx, db.InsertOrderParams{P1: "Charlie", P2: int32(7500)}); err != nil {
		fmt.Fprintln(os.Stderr, "InsertOrder 3:", err)
		os.Exit(1)
	}

	orders, err := queries.ListOrders(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListOrders:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"list_orders": orders,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
