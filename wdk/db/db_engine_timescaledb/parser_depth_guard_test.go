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

package db_engine_timescaledb

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func applyDepthGuardCase(t *testing.T, sql string) (resultError error) {
	t.Helper()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("capture loop panicked instead of returning a bounded error: %v", recovered)
		}
	}()

	engine := NewTimescaleDBEngine()
	statements, parseError := engine.ParseStatements(sql)
	if parseError != nil {
		return parseError
	}
	require.NotEmpty(t, statements, "expected at least one parsed statement")

	_, applyError := engine.ApplyDDL(context.Background(), statements[0])
	return applyError
}

func TestCaptureLoops_DeeplyNestedBodyBoundsWithDepthError(t *testing.T) {
	t.Parallel()

	overflowDepth := maxParenDepth + 8
	openers := strings.Repeat("(", overflowDepth)
	closers := strings.Repeat(")", overflowDepth)

	cases := []struct {
		description string
		sql         string
	}{
		{
			description: "create_hypertable call extras",
			sql:         "SELECT create_hypertable('readings', 'ts', x => " + openers + "1" + closers + ")",
		},
		{
			description: "continuous aggregate view body",
			sql: "CREATE MATERIALIZED VIEW v WITH (timescaledb.continuous = true) AS SELECT " +
				openers + "1" + closers,
		},
		{
			description: "keyword create hypertable check clause",
			sql:         "CREATE HYPERTABLE t (val INTEGER CHECK " + openers + "val > 0" + closers + ", ts TIMESTAMPTZ)",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.description, func(t *testing.T) {
			t.Parallel()

			err := applyDepthGuardCase(t, testCase.sql)
			require.Error(t, err, "deeply nested body must produce an error")
			assert.True(t, errors.Is(err, errParenDepthExceeded),
				"deeply nested body must return errParenDepthExceeded, got %v", err)
		})
	}
}

func TestCaptureLoops_UnterminatedBodyBoundsWithError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description    string
		sql            string
		messageContain string
	}{
		{
			description:    "create_hypertable call",
			sql:            "SELECT create_hypertable('readings', 'ts', x => (1",
			messageContain: "unterminated create_hypertable call",
		},
		{
			description:    "continuous aggregate view body",
			sql:            "CREATE MATERIALIZED VIEW v WITH (timescaledb.continuous = true) AS SELECT (1",
			messageContain: "unterminated parenthesis",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.description, func(t *testing.T) {
			t.Parallel()

			err := applyDepthGuardCase(t, testCase.sql)
			require.Error(t, err, "unterminated body must produce an error")
			assert.Contains(t, err.Error(), testCase.messageContain,
				"unterminated body must report a bounded error, got %v", err)
		})
	}
}
