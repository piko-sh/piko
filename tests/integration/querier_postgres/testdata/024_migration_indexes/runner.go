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

	_, err = conn.ExecContext(ctx, `INSERT INTO products (sku, name, category, price, active, attributes) VALUES
		('WDG-001', 'Widget A', 'widgets', 1000, true, '{"colour": "red", "weight": 100}'),
		('WDG-002', 'Widget B', 'widgets', 1500, true, '{"colour": "blue", "weight": 150}'),
		('GDG-001', 'Gadget X', 'gadgets', 2500, true, '{"colour": "red", "weight": 200}'),
		('WDG-003', 'Widget C', 'widgets', 800, false, '{"colour": "green", "weight": 120}')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	bySku, err := queries.GetProductBySku(ctx, "WDG-002")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	activeWidgets, err := queries.ListActiveByCategory(ctx, "widgets")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	byAttributes, err := queries.FindByAttributes(ctx, `{"colour": "red"}`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := map[string]any{
		"by_sku":         bySku,
		"active_widgets": activeWidgets,
		"by_attributes":  byAttributes,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
