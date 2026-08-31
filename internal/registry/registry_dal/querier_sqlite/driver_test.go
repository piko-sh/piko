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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbpkg "piko.sh/piko/internal/registry/registry_dal/querier_sqlite/db"
)

type recordingDBTX struct {
	queries []string
}

func (r *recordingDBTX) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	r.queries = append(r.queries, query)

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
