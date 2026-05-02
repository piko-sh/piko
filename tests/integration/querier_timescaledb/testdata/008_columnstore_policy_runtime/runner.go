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

	baseline, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	samples := []struct {
		offsetHours int
		device      string
		value       float64
	}{
		{0, "alpha", 10.0},
		{1, "alpha", 12.0},
		{2, "alpha", 14.0},
		{0, "beta", 5.0},
		{1, "beta", 7.0},
	}
	for _, sample := range samples {
		err = queries.InsertMetric(ctx, db.InsertMetricParams{
			Ts:     baseline.Add(time.Duration(sample.offsetHours) * time.Hour),
			Device: sample.device,
			Value:  sample.value,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	rows, err := queries.SumByDevice(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	type sumEntry struct {
		Device string  `json:"device"`
		Total  float64 `json:"total"`
	}
	out := []sumEntry{}
	for _, row := range rows {
		out = append(out, sumEntry{Device: row.Device, Total: row.Total})
	}

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"sum_by_device": out,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
