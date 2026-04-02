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
	priceAsc, err := queries.ListProductsSorted(ctx, db.ListProductsSortedParams{
		OrderBy:          db.ListProductsSortedOrderByPrice,
		OrderByDirection: db.OrderAsc,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListProductsSorted (price ASC):", err)
		os.Exit(1)
	}
	priceDesc, err := queries.ListProductsSorted(ctx, db.ListProductsSortedParams{
		OrderBy:          db.ListProductsSortedOrderByPrice,
		OrderByDirection: db.OrderDesc,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListProductsSorted (price DESC):", err)
		os.Exit(1)
	}
	nameAsc, err := queries.ListProductsSorted(ctx, db.ListProductsSortedParams{
		OrderBy:          db.ListProductsSortedOrderByName,
		OrderByDirection: db.OrderAsc,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListProductsSorted (name ASC):", err)
		os.Exit(1)
	}
	page1, err := queries.ListProductsPaginated(ctx, db.ListProductsPaginatedParams{
		PageSize:   3,
		PageOffset: 0,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListProductsPaginated (page 1):", err)
		os.Exit(1)
	}
	page2, err := queries.ListProductsPaginated(ctx, db.ListProductsPaginatedParams{
		PageSize:   3,
		PageOffset: 3,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListProductsPaginated (page 2):", err)
		os.Exit(1)
	}
	defaults, err := queries.ListProductsPaginated(ctx, db.ListProductsPaginatedParams{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListProductsPaginated (defaults):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"price_asc":  priceAsc,
		"price_desc": priceDesc,
		"name_asc":   nameAsc,
		"page1":      page1,
		"page2":      page2,
		"defaults":   defaults,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
