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

	_, err = conn.ExecContext(ctx, `CREATE TABLE contacts (
		id INTEGER PRIMARY KEY,
		name VARCHAR NOT NULL,
		address STRUCT(street VARCHAR, city VARCHAR, zip VARCHAR)
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO contacts VALUES
		(1, 'Alice', {'street': '123 Main St', 'city': 'NYC', 'zip': '10001'}),
		(2, 'Bob', {'street': '456 Oak Ave', 'city': 'LA', 'zip': '90001'}),
		(3, 'Charlie', {'street': '789 Elm Dr', 'city': 'NYC', 'zip': '10002'})`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	contact, err := queries.GetContact(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetContact:", err)
		os.Exit(1)
	}

	byCity, err := queries.ListByCityField(ctx, "NYC")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListByCityField:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"get_contact":        contact,
		"list_by_city_field": byCity,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
