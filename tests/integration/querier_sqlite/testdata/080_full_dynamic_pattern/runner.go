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

func ptrString(s string) *string { return &s }

func main() {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, `CREATE TABLE products (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		price INTEGER NOT NULL,
		category TEXT NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO products (id, name, price, category) VALUES
		(1, 'Apple', 100, 'fruit'),
		(2, 'Banana', 50, 'fruit'),
		(3, 'Carrot', 75, 'veg'),
		(4, 'Date', 200, 'fruit'),
		(5, 'Eggplant', 150, 'veg'),
		(6, 'Fig', 120, 'fruit'),
		(7, 'Grape', 80, 'fruit'),
		(8, 'Honeydew', 300, 'fruit')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	noFilterPriceDesc, err := queries.BrowseProducts(ctx, db.BrowseProductsParams{
		OrderBy:          db.BrowseProductsOrderByPrice,
		OrderByDirection: db.OrderDesc,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "BrowseProducts (no filter, price DESC):", err)
		os.Exit(1)
	}
	fruitNameAsc, err := queries.BrowseProducts(ctx, db.BrowseProductsParams{
		Category:         ptrString("fruit"),
		OrderBy:          db.BrowseProductsOrderByName,
		OrderByDirection: db.OrderAsc,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "BrowseProducts (fruit, name ASC):", err)
		os.Exit(1)
	}
	defaults, err := queries.BrowseProducts(ctx, db.BrowseProductsParams{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "BrowseProducts (defaults):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"no_filter_price_desc": noFilterPriceDesc,
		"fruit_name_asc":      fruitNameAsc,
		"defaults":            defaults,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
