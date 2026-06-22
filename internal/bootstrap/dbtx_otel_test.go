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

package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
)

type stubDBTX struct {
	ExecContextFunc     func(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContextFunc    func(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContextFunc func(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *stubDBTX) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if s.ExecContextFunc != nil {
		return s.ExecContextFunc(ctx, query, args...)
	}
	return nil, nil
}

func (s *stubDBTX) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if s.QueryContextFunc != nil {
		return s.QueryContextFunc(ctx, query, args...)
	}
	return nil, nil
}

func (s *stubDBTX) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if s.QueryRowContextFunc != nil {
		return s.QueryRowContextFunc(ctx, query, args...)
	}
	return nil
}

func TestOTelDBTX_ExecContext_DelegatesToInner(t *testing.T) {
	t.Parallel()
	called := false
	inner := &stubDBTX{
		ExecContextFunc: func(_ context.Context, _ string, _ ...any) (sql.Result, error) {
			called = true
			return nil, nil
		},
	}

	wrapper := newOTelDBTX(inner, "sqlite", "testdb", nil, nil)
	_, err := wrapper.ExecContext(context.Background(), "INSERT INTO t VALUES (?)", 1)

	require.NoError(t, err)
	assert.True(t, called)
}

func TestOTelDBTX_ExecContext_PropagatesError(t *testing.T) {
	t.Parallel()
	expectedError := errors.New("exec failed")
	inner := &stubDBTX{
		ExecContextFunc: func(_ context.Context, _ string, _ ...any) (sql.Result, error) {
			return nil, expectedError
		},
	}

	wrapper := newOTelDBTX(inner, "sqlite", "testdb", nil, nil)
	_, err := wrapper.ExecContext(context.Background(), "INSERT INTO t VALUES (?)", 1)

	assert.ErrorIs(t, err, expectedError)
}

func TestOTelDBTX_QueryContext_DelegatesToInner(t *testing.T) {
	t.Parallel()
	called := false
	inner := &stubDBTX{
		QueryContextFunc: func(_ context.Context, _ string, _ ...any) (*sql.Rows, error) {
			called = true
			return nil, nil
		},
	}

	wrapper := newOTelDBTX(inner, "postgresql", "analytics", nil, nil)
	_, err := wrapper.QueryContext(context.Background(), "SELECT 1")

	require.NoError(t, err)
	assert.True(t, called)
}

func TestOTelDBTX_QueryRowContext_DelegatesToInner(t *testing.T) {
	t.Parallel()
	called := false
	inner := &stubDBTX{
		QueryRowContextFunc: func(_ context.Context, _ string, _ ...any) *sql.Row {
			called = true
			return nil
		},
	}

	wrapper := newOTelDBTX(inner, "mysql", "users", nil, nil)
	wrapper.QueryRowContext(context.Background(), "SELECT 1")

	assert.True(t, called)
}

func TestOTelDBTX_ResolveOperation_WithResolver(t *testing.T) {
	t.Parallel()
	resolver := func(query string) string {
		if query == "SELECT * FROM tasks" {
			return "ListTasks"
		}
		return ""
	}

	wrapper := newOTelDBTX(&stubDBTX{}, "sqlite", "testdb", resolver, nil)

	assert.Equal(t, "ListTasks", wrapper.resolveOperation("SELECT * FROM tasks"))
	assert.Equal(t, "UNKNOWN", wrapper.resolveOperation("SELECT * FROM other"))
}

func TestOTelDBTX_ResolveOperation_NilResolver(t *testing.T) {
	t.Parallel()
	wrapper := newOTelDBTX(&stubDBTX{}, "sqlite", "testdb", nil, nil)

	assert.Equal(t, "UNKNOWN", wrapper.resolveOperation("SELECT 1"))
}

type captureObserver struct {
	obs []monitoring_domain.QueryObservation
}

func (c *captureObserver) ObserveQuery(_ context.Context, o *monitoring_domain.QueryObservation) {
	c.obs = append(c.obs, *o)
}

type panicObserver struct{}

func (panicObserver) ObserveQuery(context.Context, *monitoring_domain.QueryObservation) {
	panic("observer boom")
}

func TestOTelDBTX_Observe_PanicRecovered(t *testing.T) {
	t.Parallel()
	inner := &stubDBTX{
		ExecContextFunc: func(_ context.Context, _ string, _ ...any) (sql.Result, error) {
			return fakeResult{rows: 1}, nil
		},
	}
	wrapper := newOTelDBTX(inner, "sqlite", "testdb", nil, panicObserver{})
	require.NotPanics(t, func() {
		result, err := wrapper.ExecContext(context.Background(), "UPDATE t SET x = ?", 1)
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

type fakeResult struct{ rows int64 }

func (f fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (f fakeResult) RowsAffected() (int64, error) { return f.rows, nil }

func TestOTelDBTX_Observe_ExecRowsAffected(t *testing.T) {
	t.Parallel()
	obs := &captureObserver{}
	inner := &stubDBTX{
		ExecContextFunc: func(_ context.Context, _ string, _ ...any) (sql.Result, error) {
			return fakeResult{rows: 7}, nil
		},
	}
	wrapper := newOTelDBTX(inner, "sqlite", "testdb", nil, obs)
	_, err := wrapper.ExecContext(context.Background(), "UPDATE t SET x = ?", 1)
	require.NoError(t, err)
	require.Len(t, obs.obs, 1)
	assert.Equal(t, int64(7), obs.obs[0].Rows)
	assert.Equal(t, "testdb", obs.obs[0].Connection)
	assert.Equal(t, "sqlite", obs.obs[0].System)
	assert.NoError(t, obs.obs[0].Err)
}

func TestOTelDBTX_Observe_QueryRowsZero(t *testing.T) {
	t.Parallel()
	obs := &captureObserver{}
	wrapper := newOTelDBTX(&stubDBTX{}, "postgresql", "analytics", nil, obs)
	_, _ = wrapper.QueryContext(context.Background(), "SELECT 1")
	require.Len(t, obs.obs, 1)
	assert.Equal(t, int64(0), obs.obs[0].Rows)
	assert.Equal(t, "postgresql", obs.obs[0].System)
}

func TestOTelDBTX_Observe_NilObserverNoPanic(t *testing.T) {
	t.Parallel()
	wrapper := newOTelDBTX(&stubDBTX{}, "sqlite", "testdb", nil, nil)
	_, err := wrapper.ExecContext(context.Background(), "SELECT 1")
	require.NoError(t, err)
}

func TestResolveDBSystem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		driverName     string
		engineDriver   string
		expectedSystem string
	}{
		{driverName: "postgres", expectedSystem: "postgresql"},
		{driverName: "pgx", expectedSystem: "postgresql"},
		{driverName: "mysql", expectedSystem: "mysql"},
		{driverName: "sqlite", expectedSystem: "sqlite"},
		{driverName: "sqlite3", expectedSystem: "sqlite"},
		{driverName: "duckdb", expectedSystem: "duckdb"},
		{driverName: "custom", expectedSystem: "custom"},
		{engineDriver: "postgres", expectedSystem: "postgresql"},
		{driverName: "", engineDriver: "", expectedSystem: ""},
	}

	for _, test := range tests {
		registration := &DatabaseRegistration{
			DriverName: test.driverName,
			EngineConfig: EngineConfig{
				DriverName: test.engineDriver,
			},
		}
		result := resolveDBSystem(registration)
		assert.Equal(t, test.expectedSystem, result, "driver=%q engine=%q", test.driverName, test.engineDriver)
	}
}
