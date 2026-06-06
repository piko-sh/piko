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
	"math"
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

	cohortA := []float64{9.5, 10.1, 10.0, 9.8, 10.2, 9.9, 10.3, 10.0, 9.7, 10.1}
	cohortB := []float64{11.8, 12.1, 12.0, 12.2, 11.9, 12.3, 12.0, 12.1, 11.7, 12.2}
	for _, value := range cohortA {
		if err := queries.InsertSample(ctx, db.InsertSampleParams{Cohort: "a", Value: value}); err != nil {
			return fmt.Errorf("insert cohort a: %w", err)
		}
	}
	for _, value := range cohortB {
		if err := queries.InsertSample(ctx, db.InsertSampleParams{Cohort: "b", Value: value}); err != nil {
			return fmt.Errorf("insert cohort b: %w", err)
		}
	}

	row, err := queries.WelchTest(ctx)
	if err != nil {
		return fmt.Errorf("welch test: %w", err)
	}

	tStat, ok := row.TStatistic.(float64)
	if !ok {
		return fmt.Errorf("t_statistic was %T, expected float64", row.TStatistic)
	}
	pValue, ok := row.PValue.(float64)
	if !ok {
		return fmt.Errorf("p_value was %T, expected float64", row.PValue)
	}

	round := func(value float64) float64 {
		return math.Round(value*1000) / 1000
	}

	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"t_statistic_sign":    math.Signbit(tStat),
		"t_statistic_rounded": round(tStat),
		"p_value_below_0p001": pValue < 0.001,
	})
}
