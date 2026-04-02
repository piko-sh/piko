package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"querier_test_runner/db"
)

func main() {
	connectionString := os.Getenv("DATABASE_URL")
	schemaName := os.Getenv("DATABASE_SCHEMA")

	conn, err := sql.Open("pgx", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schemaName))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO products (name, price, category) VALUES
		('Apple', 100, 'fruit'),
		('Banana', 50, 'fruit'),
		('Carrot', 75, 'veg'),
		('Date', 200, 'fruit'),
		('Eggplant', 150, 'veg')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	sortedByNameAsc, err := queries.ListProductsSorted(ctx, db.ListProductsSortedParams{
		OrderBy:          db.ListProductsSortedOrderByName,
		OrderByDirection: db.OrderAsc,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListProductsSorted (name ASC):", err)
		os.Exit(1)
	}
	sortedByPriceDesc, err := queries.ListProductsSorted(ctx, db.ListProductsSortedParams{
		OrderBy:          db.ListProductsSortedOrderByPrice,
		OrderByDirection: db.OrderDesc,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListProductsSorted (price DESC):", err)
		os.Exit(1)
	}
	unsorted, err := queries.ListProductsSorted(ctx, db.ListProductsSortedParams{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListProductsSorted (unsorted):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"sorted_by_name_asc":  sortedByNameAsc,
		"sorted_by_price_desc": sortedByPriceDesc,
		"unsorted":             unsorted,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
