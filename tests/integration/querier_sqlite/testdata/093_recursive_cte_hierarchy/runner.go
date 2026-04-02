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

	_, err = conn.ExecContext(ctx, `CREATE TABLE categories (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		parent_id INTEGER REFERENCES categories(id)
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO categories (id, name, parent_id) VALUES
		(1, 'Root', NULL),
		(2, 'Electronics', 1),
		(3, 'Clothing', 1),
		(4, 'Phones', 2),
		(5, 'Laptops', 2),
		(6, 'T-Shirts', 3),
		(7, 'iPhones', 4)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
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

	result := map[string]any{
		"subtree_from_electronics": subtree,
		"ancestors_of_iphones":    ancestors,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
