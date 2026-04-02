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

	wrapper := newOTelDBTX(inner, "sqlite", "testdb", nil)
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

	wrapper := newOTelDBTX(inner, "sqlite", "testdb", nil)
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

	wrapper := newOTelDBTX(inner, "postgresql", "analytics", nil)
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

	wrapper := newOTelDBTX(inner, "mysql", "users", nil)
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

	wrapper := newOTelDBTX(&stubDBTX{}, "sqlite", "testdb", resolver)

	assert.Equal(t, "ListTasks", wrapper.resolveOperation("SELECT * FROM tasks"))
	assert.Equal(t, "UNKNOWN", wrapper.resolveOperation("SELECT * FROM other"))
}

func TestOTelDBTX_ResolveOperation_NilResolver(t *testing.T) {
	t.Parallel()
	wrapper := newOTelDBTX(&stubDBTX{}, "sqlite", "testdb", nil)

	assert.Equal(t, "UNKNOWN", wrapper.resolveOperation("SELECT 1"))
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
