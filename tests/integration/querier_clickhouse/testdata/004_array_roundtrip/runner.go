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
	mustErr(queries.Insert(ctx, db.InsertParams{ID: 1, Tags: []string{"red", "green", "blue"}}))
	row, err := queries.Get(ctx, 1)
	mustErr(err)
	_ = json.NewEncoder(os.Stdout).Encode(row)
}
