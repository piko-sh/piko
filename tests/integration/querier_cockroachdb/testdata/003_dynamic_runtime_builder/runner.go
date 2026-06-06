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
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	connectionString := os.Getenv("DATABASE_URL")
	schemaName := os.Getenv("DATABASE_SCHEMA")

	conn, err := sql.Open("pgx", connectionString)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx := context.Background()

	if _, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schemaName)); err != nil {
		return err
	}

	queries := db.New(conn)

	products := []db.InsertProductParams{
		{Name: "widget", Price: 100, InStock: true},
		{Name: "gadget", Price: 150, InStock: false},
		{Name: "gizmo", Price: 200, InStock: true},
		{Name: "gimbal", Price: 250, InStock: true},
	}
	for index, product := range products {
		if err = queries.InsertProduct(ctx, product); err != nil {
			return fmt.Errorf("insert product %d: %w", index, err)
		}
	}

	expensive, err := queries.SearchProducts(ctx).
		Where("price", ">=", int64(150)).
		OrderBy("id", "ASC").
		Limit(2).
		All(ctx)
	if err != nil {
		return fmt.Errorf("dynamic search: %w", err)
	}

	stocked, err := queries.SearchStocked(ctx, true).
		Where("price", ">=", int64(150)).
		OrderBy("id", "ASC").
		All(ctx)
	if err != nil {
		return fmt.Errorf("dynamic search with static param: %w", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]any{
		"expensive_products":         expensive,
		"stocked_expensive_products": stocked,
	})
}
