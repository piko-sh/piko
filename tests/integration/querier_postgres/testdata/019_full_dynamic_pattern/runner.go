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

func ptrString(s string) *string { return &s }

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
		('Eggplant', 150, 'veg'),
		('Fig', 120, 'fruit'),
		('Grape', 80, 'fruit'),
		('Honeydew', 300, 'fruit')`)
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
