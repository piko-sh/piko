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

	const baseEpoch = int64(1_767_312_000)
	samples := []struct {
		offsetSeconds int64
		sensor        int64
		value         float64
	}{
		{0, 1, 1.0},
		{30, 1, 2.0},
		{59, 1, 3.0},
		{60, 1, 4.0},
		{120, 1, 5.0},
	}
	for _, sample := range samples {
		err = queries.InsertIntegerMetric(ctx, db.InsertIntegerMetricParams{
			TsEpoch: baseEpoch + sample.offsetSeconds,
			Sensor:  sample.sensor,
			Value:   sample.value,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	rows, err := queries.PerMinuteSum(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	type bucketRow struct {
		Bucket int64   `json:"bucket"`
		Total  float64 `json:"total"`
	}
	out := []bucketRow{}
	for _, row := range rows {
		out = append(out, bucketRow{Bucket: row.Bucket, Total: row.Total})
	}

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"per_minute": out,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
