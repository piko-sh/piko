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

func stringPointer(s string) *string { return &s }

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
	emptyBatch := []db.InsertItemsBatchParams{}
	err = queries.InsertItemsBatch(ctx, emptyBatch)
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertItemsBatch (empty):", err)
		os.Exit(1)
	}

	emptyCount, err := queries.CountItems(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountItems (empty):", err)
		os.Exit(1)
	}
	singleBatch := []db.InsertItemsBatchParams{
		{P1: int32(1), P2: "Solo", P3: "misc", P4: int32(999), P5: nil},
	}
	err = queries.InsertItemsBatch(ctx, singleBatch)
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertItemsBatch (single):", err)
		os.Exit(1)
	}

	singleItem, err := queries.GetItem(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetItem (single):", err)
		os.Exit(1)
	}
	_, err = conn.ExecContext(ctx, "DELETE FROM items")
	if err != nil {
		fmt.Fprintln(os.Stderr, "DELETE (before small):", err)
		os.Exit(1)
	}
	smallBatch := []db.InsertItemsBatchParams{
		{P1: int32(1), P2: "Apple", P3: "fruit", P4: int32(100), P5: stringPointer("A crisp fruit")},
		{P1: int32(2), P2: "Banana", P3: "fruit", P4: int32(50), P5: nil},
		{P1: int32(3), P2: "Carrot", P3: "veg", P4: int32(75), P5: stringPointer("An orange root")},
	}
	err = queries.InsertItemsBatch(ctx, smallBatch)
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertItemsBatch (small):", err)
		os.Exit(1)
	}

	smallItems, err := queries.ListItems(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListItems (small):", err)
		os.Exit(1)
	}
	_, err = conn.ExecContext(ctx, "DELETE FROM items")
	if err != nil {
		fmt.Fprintln(os.Stderr, "DELETE (before boundary):", err)
		os.Exit(1)
	}
	boundarySize := 199
	boundaryBatch := make([]db.InsertItemsBatchParams, boundarySize)
	for i := range boundaryBatch {
		boundaryBatch[i] = db.InsertItemsBatchParams{
			P1: int32(i + 1),
			P2: fmt.Sprintf("item_%d", i+1),
			P3: "bulk",
			P4: int32(i * 10),
			P5: stringPointer(fmt.Sprintf("desc_%d", i+1)),
		}
	}
	err = queries.InsertItemsBatch(ctx, boundaryBatch)
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertItemsBatch (boundary):", err)
		os.Exit(1)
	}

	boundaryCount, err := queries.CountItems(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountItems (boundary):", err)
		os.Exit(1)
	}
	_, err = conn.ExecContext(ctx, "DELETE FROM items")
	if err != nil {
		fmt.Fprintln(os.Stderr, "DELETE (before boundary+1):", err)
		os.Exit(1)
	}
	overBoundarySize := 200
	overBoundaryBatch := make([]db.InsertItemsBatchParams, overBoundarySize)
	for i := range overBoundaryBatch {
		overBoundaryBatch[i] = db.InsertItemsBatchParams{
			P1: int32(i + 1),
			P2: fmt.Sprintf("item_%d", i+1),
			P3: "bulk",
			P4: int32(i * 10),
			P5: stringPointer(fmt.Sprintf("desc_%d", i+1)),
		}
	}
	err = queries.InsertItemsBatch(ctx, overBoundaryBatch)
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertItemsBatch (boundary+1):", err)
		os.Exit(1)
	}

	overBoundaryCount, err := queries.CountItems(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountItems (boundary+1):", err)
		os.Exit(1)
	}
	overBoundaryLastItem, err := queries.GetItem(ctx, int32(200))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetItem (boundary+1 last):", err)
		os.Exit(1)
	}
	_, err = conn.ExecContext(ctx, "DELETE FROM items")
	if err != nil {
		fmt.Fprintln(os.Stderr, "DELETE (before large):", err)
		os.Exit(1)
	}
	largeSize := 300
	largeBatch := make([]db.InsertItemsBatchParams, largeSize)
	for i := range largeBatch {
		largeBatch[i] = db.InsertItemsBatchParams{
			P1: int32(i + 1),
			P2: fmt.Sprintf("item_%d", i+1),
			P3: "bulk",
			P4: int32(i * 10),
			P5: stringPointer(fmt.Sprintf("desc_%d", i+1)),
		}
	}
	err = queries.InsertItemsBatch(ctx, largeBatch)
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertItemsBatch (large):", err)
		os.Exit(1)
	}

	largeCount, err := queries.CountItems(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountItems (large):", err)
		os.Exit(1)
	}
	largeFirstItem, err := queries.GetItem(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetItem (large first):", err)
		os.Exit(1)
	}

	largeLastItem, err := queries.GetItem(ctx, int32(300))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetItem (large last):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"01_empty_count":             emptyCount,
		"02_single_item":             singleItem,
		"03_small_items":             smallItems,
		"04_boundary_count":          boundaryCount,
		"05_over_boundary_count":     overBoundaryCount,
		"06_over_boundary_last_item": overBoundaryLastItem,
		"07_large_count":             largeCount,
		"08_large_first_item":        largeFirstItem,
		"09_large_last_item":         largeLastItem,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
