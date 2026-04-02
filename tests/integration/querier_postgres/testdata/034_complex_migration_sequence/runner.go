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

	_, err = conn.ExecContext(ctx, `INSERT INTO customers (name, email, metadata) VALUES
		('Alice', 'alice@example.com', '{"region": "eu"}'),
		('Bob', 'bob@example.com', '{"region": "us"}')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	noteText := "Express delivery requested"
	order1, err := queries.InsertOrder(ctx, db.InsertOrderParams{
		P1: int32(1),
		P2: int32(5000),
		P3: "confirmed",
		P4: &noteText,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	order2, err := queries.InsertOrder(ctx, db.InsertOrderParams{
		P1: int32(1),
		P2: int32(3000),
		P3: "pending",
		P4: nil,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	order3, err := queries.InsertOrder(ctx, db.InsertOrderParams{
		P1: int32(2),
		P2: int32(7500),
		P3: "shipped",
		P4: nil,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	customer, err := queries.GetCustomerByEmail(ctx, "alice@example.com")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	summary, err := queries.GetOrderSummary(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := map[string]any{
		"order_1":  order1,
		"order_2":  order2,
		"order_3":  order3,
		"customer": customer,
		"summary":  summary,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
