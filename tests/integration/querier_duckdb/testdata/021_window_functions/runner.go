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

	_, err = conn.ExecContext(ctx, `CREATE TABLE sales (
		id INTEGER PRIMARY KEY,
		employee VARCHAR NOT NULL,
		amount INTEGER NOT NULL,
		sale_date VARCHAR NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO sales (id, employee, amount, sale_date) VALUES
		(1, 'Alice', 100, '2026-01-01'),
		(2, 'Bob', 200, '2026-01-02'),
		(3, 'Alice', 150, '2026-01-03'),
		(4, 'Bob', 300, '2026-01-04'),
		(5, 'Charlie', 250, '2026-01-05'),
		(6, 'Charlie', 175, '2026-01-06')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	runningTotal, err := queries.ListWithRunningTotal(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListWithRunningTotal:", err)
		os.Exit(1)
	}

	lagLead, err := queries.ListWithLagLead(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListWithLagLead:", err)
		os.Exit(1)
	}

	topSale, err := queries.TopSalePerEmployee(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "TopSalePerEmployee:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"list_with_running_total": runningTotal,
		"list_with_lag_lead":      lagLead,
		"top_sale_per_employee":   topSale,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
