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

	_, err = conn.ExecContext(ctx, `CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT NOT NULL, category TEXT NOT NULL, price INTEGER NOT NULL, sku TEXT NOT NULL)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE INDEX idx_products_category ON products (category)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE UNIQUE INDEX idx_products_sku ON products (sku)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE INDEX idx_products_category_price ON products (category, price)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO products (id, name, category, price, sku) VALUES
		(1, 'Laptop', 'electronics', 999, 'ELEC-001'),
		(2, 'Phone', 'electronics', 699, 'ELEC-002'),
		(3, 'Tablet', 'electronics', 399, 'ELEC-003'),
		(4, 'Desk', 'furniture', 250, 'FURN-001'),
		(5, 'Chair', 'furniture', 150, 'FURN-002'),
		(6, 'Monitor', 'electronics', 349, 'ELEC-004')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	byCategory, err := queries.ListByCategory(ctx, "electronics")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListByCategory:", err)
		os.Exit(1)
	}

	bySku, err := queries.GetBySku(ctx, "FURN-001")
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetBySku:", err)
		os.Exit(1)
	}

	byCategoryAndPrice, err := queries.ListByCategoryAndMaxPrice(ctx, db.ListByCategoryAndMaxPriceParams{
		P1: "electronics",
		P2: int32(500),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListByCategoryAndMaxPrice:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"by_category":           byCategory,
		"by_sku":                bySku,
		"by_category_and_price": byCategoryAndPrice,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
