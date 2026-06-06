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
	"github.com/google/uuid"

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
	id := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	mustErr(queries.Insert(ctx, db.InsertParams{ID: id, Label: "session-a"}))
	row, err := queries.Get(ctx, id)
	mustErr(err)
	_ = json.NewEncoder(os.Stdout).Encode(row)
}
