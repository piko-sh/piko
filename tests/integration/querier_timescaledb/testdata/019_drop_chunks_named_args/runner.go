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
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"querier_test_runner/db"
)

func main() {
	connectionString := os.Getenv("DATABASE_URL")
	schemaName := os.Getenv("DATABASE_SCHEMA")

	conn, err := sql.Open("pgx", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, public", schemaName))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	old := time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2026, time.May, 30, 0, 0, 0, 0, time.UTC)
	seeds := []struct {
		ts    time.Time
		value float64
	}{
		{old, 1.0},
		{old.Add(24 * time.Hour), 2.0},
		{old.Add(48 * time.Hour), 3.0},
		{recent, 10.0},
		{recent.Add(time.Hour), 11.0},
	}
	for _, seed := range seeds {
		err = queries.InsertTelemetry(ctx, db.InsertTelemetryParams{
			Ts:    seed.ts,
			Value: seed.value,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	dropped, err := queries.DropOldChunks(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	remaining, err := queries.RemainingRows(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out := []map[string]any{}
	for _, name := range dropped {
		out = append(out, map[string]any{"ChunkName": name})
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"dropped":   out,
		"remaining": remaining,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
