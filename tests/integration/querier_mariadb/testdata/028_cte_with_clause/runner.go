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

	entries := []db.InsertCategoryParams{
		{P1: int32(1), P2: "Root", P3: sql.NullInt32{Valid: false}},
		{P1: int32(2), P2: "Electronics", P3: sql.NullInt32{Int32: 1, Valid: true}},
		{P1: int32(3), P2: "Clothing", P3: sql.NullInt32{Int32: 1, Valid: true}},
		{P1: int32(4), P2: "Phones", P3: sql.NullInt32{Int32: 2, Valid: true}},
		{P1: int32(5), P2: "Laptops", P3: sql.NullInt32{Int32: 2, Valid: true}},
		{P1: int32(6), P2: "T-Shirts", P3: sql.NullInt32{Int32: 3, Valid: true}},
		{P1: int32(7), P2: "iPhones", P3: sql.NullInt32{Int32: 4, Valid: true}},
	}

	for i, e := range entries {
		err = queries.InsertCategory(ctx, e)
		if err != nil {
			fmt.Fprintf(os.Stderr, "InsertCategory %d: %v\n", i+1, err)
			os.Exit(1)
		}
	}
	subtree, err := queries.GetSubtree(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetSubtree:", err)
		os.Exit(1)
	}
	ancestors, err := queries.GetAncestors(ctx, int32(7))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetAncestors:", err)
		os.Exit(1)
	}
	roots, err := queries.ListRootCategories(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListRootCategories:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"subtree_from_electronics": subtree,
		"ancestors_of_iphones":    ancestors,
		"root_categories":         roots,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
