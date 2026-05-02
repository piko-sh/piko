// Copyright 2026 PolitePixels Limited
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/ClickHouse/clickhouse-go/v2"

	"querier_clickhouse_runner/db"
)

func mustErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	conn, err := sql.Open("clickhouse", os.Getenv("DATABASE_URL"))
	mustErr(err)
	defer conn.Close()
	ctx := context.Background()
	queries := db.New(conn)
	mustErr(queries.InsertSale(ctx, db.InsertSaleParams{ID: 1, Region: "EU", Amount: 100}))
	mustErr(queries.InsertSale(ctx, db.InsertSaleParams{ID: 2, Region: "EU", Amount: 200}))
	mustErr(queries.InsertSale(ctx, db.InsertSaleParams{ID: 3, Region: "US", Amount: 50}))
	rows, err := queries.TotalByRegion(ctx)
	mustErr(err)
	_ = json.NewEncoder(os.Stdout).Encode(rows)
}
