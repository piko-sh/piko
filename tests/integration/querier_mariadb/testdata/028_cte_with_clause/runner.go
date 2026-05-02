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

	parentPtr := func(value int32) *int32 { return &value }

	entries := []db.InsertCategoryParams{
		{ID: int32(1), Name: "Root", ParentID: nil},
		{ID: int32(2), Name: "Electronics", ParentID: parentPtr(1)},
		{ID: int32(3), Name: "Clothing", ParentID: parentPtr(1)},
		{ID: int32(4), Name: "Phones", ParentID: parentPtr(2)},
		{ID: int32(5), Name: "Laptops", ParentID: parentPtr(2)},
		{ID: int32(6), Name: "T-Shirts", ParentID: parentPtr(3)},
		{ID: int32(7), Name: "iPhones", ParentID: parentPtr(4)},
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
		"ancestors_of_iphones":     ancestors,
		"root_categories":          roots,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
