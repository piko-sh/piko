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

	_, err = conn.ExecContext(ctx, `CREATE TABLE categories (
		id INTEGER PRIMARY KEY,
		name VARCHAR NOT NULL,
		parent_id INTEGER
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO categories (id, name, parent_id) VALUES
		(1, 'Electronics', NULL),
		(2, 'Computers', 1),
		(3, 'Laptops', 2),
		(4, 'Desktops', 2),
		(5, 'Phones', 1)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	subtree, err := queries.GetSubtree(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetSubtree:", err)
		os.Exit(1)
	}

	ancestors, err := queries.GetAncestors(ctx, int32(3))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetAncestors:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"get_subtree":   subtree,
		"get_ancestors": ancestors,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
