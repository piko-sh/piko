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

func main() {
	conn, err := sql.Open("clickhouse", os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()
	ctx := context.Background()
	queries := db.New(conn)
	out, err := queries.ReadMV(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(out)
}
