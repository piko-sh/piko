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

package querier_sqlite

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dal/dalcore"
	dbpkg "piko.sh/piko/internal/registry/registry_dal/querier_sqlite/db"
)

type recordingDBTX struct {
	queries   []string
	arguments [][]any
}

func (r *recordingDBTX) ExecContext(_ context.Context, query string, arguments ...any) (sql.Result, error) {
	r.queries = append(r.queries, query)
	r.arguments = append(r.arguments, arguments)

	return nil, sql.ErrConnDone
}

func (r *recordingDBTX) QueryContext(_ context.Context, query string, _ ...any) (*sql.Rows, error) {
	r.queries = append(r.queries, query)

	return nil, sql.ErrConnDone
}

func (r *recordingDBTX) QueryRowContext(_ context.Context, query string, _ ...any) *sql.Row {
	r.queries = append(r.queries, query)

	return nil
}

func TestNewObserved_RunsStatementsThroughTheObservedHandle(t *testing.T) {
	t.Parallel()

	recorder := &recordingDBTX{}
	dal := NewObserved(new(sql.DB), recorder)

	require.NotNil(t, dal, "the DAL is constructed from the two handles")
	assert.Implements(t, (*any)(nil), dal)

	queries := dbpkg.New(recorder)
	assert.NotNil(t, queries)
}

func TestDriver_InsertVariantChunkBindsNullableDuration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		duration *float64
		want     sqldriver.Value
		name     string
	}{
		{name: "nil duration is bound as NULL", duration: nil, want: nil},
		{name: "set duration is bound as its value", duration: new(2.5), want: 2.5},
		{name: "zero duration stays a value rather than NULL", duration: new(0.0), want: 0.0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recorder := &recordingDBTX{}
			chunkDriver := &driver{queries: dbpkg.New(recorder)}
			err := chunkDriver.InsertVariantChunk(t.Context(), dalcore.InsertVariantChunkParams{DurationSeconds: testCase.duration})
			require.ErrorIs(t, err, sql.ErrConnDone)

			require.Len(t, recorder.arguments, 1)
			arguments := recorder.arguments[0]
			require.NotEmpty(t, arguments)
			valuer, ok := arguments[len(arguments)-1].(sqldriver.Valuer)
			require.True(t, ok, "the duration is bound through a driver value")

			bound, err := valuer.Value()
			require.NoError(t, err)
			assert.Equal(t, testCase.want, bound)
		})
	}
}
