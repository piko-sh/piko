// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

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
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	conn, err := sql.Open("clickhouse", os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer conn.Close()

	ctx := context.Background()
	queries := db.New(conn)

	products := []db.InsertProductParams{
		{ID: 1, Name: "widget", Price: 100, InStock: true},
		{ID: 2, Name: "gadget", Price: 150, InStock: false},
		{ID: 3, Name: "gizmo", Price: 200, InStock: true},
		{ID: 4, Name: "gimbal", Price: 250, InStock: true},
	}
	for index, product := range products {
		if err := queries.InsertProduct(ctx, product); err != nil {
			return fmt.Errorf("insert product %d: %w", index, err)
		}
	}

	filtered, err := queries.SearchProducts(ctx).
		Where("price", ">=", int32(150)).
		OrderBy("id", "DESC").
		Limit(2).
		All(ctx)
	if err != nil {
		return fmt.Errorf("dynamic search: %w", err)
	}

	stocked, err := queries.SearchStocked(ctx, true).
		Where("price", ">=", int32(150)).
		OrderBy("id", "ASC").
		All(ctx)
	if err != nil {
		return fmt.Errorf("dynamic search with static param: %w", err)
	}

	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"expensive_products":         filtered,
		"stocked_expensive_products": stocked,
	})
}
