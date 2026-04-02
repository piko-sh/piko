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

	err = queries.InsertLineItem(ctx, db.InsertLineItemParams{
		P1: "Widget",
		P2: int32(5),
		P3: int32(200),
		P4: int32(0),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertLineItem 1:", err)
		os.Exit(1)
	}

	err = queries.InsertLineItem(ctx, db.InsertLineItemParams{
		P1: "Gadget",
		P2: int32(3),
		P3: int32(150),
		P4: int32(10),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertLineItem 2:", err)
		os.Exit(1)
	}

	err = queries.InsertLineItem(ctx, db.InsertLineItemParams{
		P1: "Gizmo",
		P2: int32(10),
		P3: int32(50),
		P4: int32(25),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertLineItem 3:", err)
		os.Exit(1)
	}

	item, err := queries.GetLineItem(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetLineItem:", err)
		os.Exit(1)
	}

	allItems, err := queries.ListLineItems(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListLineItems:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"item":      item,
		"all_items": allItems,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
