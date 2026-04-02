package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"querier_test_runner/db"
)

func main() {
	connectionString := os.Getenv("DATABASE_URL")

	conn, err := sql.Open("mysql", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	queries := db.New(conn)

	err = queries.InsertProduct(ctx, db.InsertProductParams{
		P1: "Laptop",
		P2: `{"price": 999, "category": "electronics", "tags": ["portable", "work"]}`,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertProduct 1:", err)
		os.Exit(1)
	}

	err = queries.InsertProduct(ctx, db.InsertProductParams{
		P1: "T-Shirt",
		P2: `{"price": 25, "category": "clothing", "tags": ["casual"]}`,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertProduct 2:", err)
		os.Exit(1)
	}

	err = queries.InsertProduct(ctx, db.InsertProductParams{
		P1: "Keyboard",
		P2: `{"price": 75, "category": "electronics", "tags": ["peripheral"]}`,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertProduct 3:", err)
		os.Exit(1)
	}

	price, err := queries.GetProductPrice(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetProductPrice:", err)
		os.Exit(1)
	}

	category, err := queries.GetProductCategory(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetProductCategory:", err)
		os.Exit(1)
	}

	electronics, err := queries.FindByCategory(ctx, "electronics")
	if err != nil {
		fmt.Fprintln(os.Stderr, "FindByCategory:", err)
		os.Exit(1)
	}

	summary, err := queries.BuildSummary(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "BuildSummary:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"price":       price,
		"category":    category,
		"electronics": electronics,
		"summary":     summary,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
