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
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/provider/provider_domain"
)

type stubConnector struct{}

func (stubConnector) Connect(_ context.Context) (driver.Conn, error) {
	return stubConn{}, nil
}

func (stubConnector) Driver() driver.Driver { return stubDriver{} }

type stubDriver struct{}

func (stubDriver) Open(_ string) (driver.Conn, error) { return stubConn{}, nil }

type stubConn struct{}

func (stubConn) Prepare(_ string) (driver.Stmt, error) { return nil, nil }
func (stubConn) Close() error                          { return nil }
func (stubConn) Begin() (driver.Tx, error)             { return nil, nil }

func newTestDB() *sql.DB {
	return sql.OpenDB(stubConnector{})
}

func newTestDatabaseService(instances map[string]*databaseInstance) *databaseService {
	return &databaseService{instances: instances}
}

func TestDatabaseService_ResourceType(t *testing.T) {
	t.Parallel()

	svc := newTestDatabaseService(nil)
	assert.Equal(t, "database", svc.ResourceType())
}

func TestDatabaseService_ResourceListColumns(t *testing.T) {
	t.Parallel()

	svc := newTestDatabaseService(nil)
	cols := svc.ResourceListColumns()
	require.Len(t, cols, 5)
	assert.Equal(t, "NAME", cols[0].Header)
	assert.Equal(t, "name", cols[0].Key)
	assert.Equal(t, "DRIVER", cols[1].Header)
	assert.Equal(t, "driver", cols[1].Key)
	assert.Equal(t, "REPLICAS", cols[2].Header)
	assert.True(t, cols[2].WideOnly)
	assert.Equal(t, "OPEN", cols[3].Header)
	assert.True(t, cols[3].WideOnly)
	assert.Equal(t, "IN USE", cols[4].Header)
	assert.True(t, cols[4].WideOnly)
}

func TestDatabaseService_ResourceListProviders(t *testing.T) {
	t.Parallel()

	t.Run("empty instances returns empty", func(t *testing.T) {
		t.Parallel()
		svc := newTestDatabaseService(map[string]*databaseInstance{})

		entries := svc.ResourceListProviders(context.Background())
		assert.Empty(t, entries)
	})

	t.Run("single database", func(t *testing.T) {
		t.Parallel()
		db := newTestDB()
		defer db.Close()

		svc := newTestDatabaseService(map[string]*databaseInstance{
			"primary": {db: db, driverName: "postgres", replicaCount: 2},
		})

		entries := svc.ResourceListProviders(context.Background())
		require.Len(t, entries, 1)
		assert.Equal(t, "primary", entries[0].Name)
		assert.Equal(t, "primary", entries[0].Values["name"])
		assert.Equal(t, "postgres", entries[0].Values["driver"])
		assert.Equal(t, "2", entries[0].Values["replicas"])
		assert.False(t, entries[0].IsDefault)
	})

	t.Run("multiple databases are sorted", func(t *testing.T) {
		t.Parallel()
		db1 := newTestDB()
		db2 := newTestDB()
		defer db1.Close()
		defer db2.Close()

		svc := newTestDatabaseService(map[string]*databaseInstance{
			"zebra": {db: db1, driverName: "mysql"},
			"alpha": {db: db2, driverName: "postgres"},
		})

		entries := svc.ResourceListProviders(context.Background())
		require.Len(t, entries, 2)
		assert.Equal(t, "alpha", entries[0].Name)
		assert.Equal(t, "zebra", entries[1].Name)
	})
}

func TestDatabaseService_ResourceDescribeProvider(t *testing.T) {
	t.Parallel()

	t.Run("not found returns error", func(t *testing.T) {
		t.Parallel()
		svc := newTestDatabaseService(map[string]*databaseInstance{})

		_, err := svc.ResourceDescribeProvider(context.Background(), "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns overview and pool sections", func(t *testing.T) {
		t.Parallel()
		db := newTestDB()
		defer db.Close()

		svc := newTestDatabaseService(map[string]*databaseInstance{
			"main": {db: db, driverName: "sqlite", replicaCount: 0},
		})

		detail, err := svc.ResourceDescribeProvider(context.Background(), "main")
		require.NoError(t, err)
		assert.Equal(t, "main", detail.Name)
		require.Len(t, detail.Sections, 2)

		assert.Equal(t, "Overview", detail.Sections[0].Title)
		assertInfoEntry(t, detail.Sections[0].Entries, "Name", "main")
		assertInfoEntry(t, detail.Sections[0].Entries, "Driver", "sqlite")
		assertInfoEntry(t, detail.Sections[0].Entries, "Replicas", "0")

		assert.Equal(t, "Connection Pool", detail.Sections[1].Title)
		assert.True(t, len(detail.Sections[1].Entries) > 0)
	})

	t.Run("includes engine diagnostics when available", func(t *testing.T) {
		t.Parallel()
		db := newTestDB()
		defer db.Close()

		svc := newTestDatabaseService(map[string]*databaseInstance{
			"diag": {
				db:         db,
				driverName: "postgres",
				engineHealthChecker: mockHealthChecker{diagnostics: []DatabaseHealthDiagnostic{
					{Name: "database_size", Value: "142 MiB", State: "HEALTHY"},
					{Name: "replication_lag", Value: "3.2s", State: "DEGRADED", Message: "lag above threshold"},
				}},
			},
		})

		detail, err := svc.ResourceDescribeProvider(context.Background(), "diag")
		require.NoError(t, err)
		require.Len(t, detail.Sections, 3)

		diagSection := detail.Sections[2]
		assert.Equal(t, "Engine Diagnostics", diagSection.Title)
		require.Len(t, diagSection.Entries, 2)
		assert.Equal(t, "database_size", diagSection.Entries[0].Key)
		assert.Equal(t, "142 MiB", diagSection.Entries[0].Value)
		assert.Equal(t, "replication_lag", diagSection.Entries[1].Key)
		assert.Equal(t, "3.2s (DEGRADED)", diagSection.Entries[1].Value)
	})
}

func TestDatabaseService_ResourceDescribeType(t *testing.T) {
	t.Parallel()

	db := newTestDB()
	defer db.Close()

	svc := newTestDatabaseService(map[string]*databaseInstance{
		"one": {db: db, driverName: "postgres"},
	})

	detail := svc.ResourceDescribeType(context.Background())
	assert.Equal(t, "database", detail.Name)
	require.Len(t, detail.Sections, 1)
	assert.Equal(t, "Overview", detail.Sections[0].Title)
	assertInfoEntry(t, detail.Sections[0].Entries, "Resource Type", "database")
	assertInfoEntry(t, detail.Sections[0].Entries, "Database Count", "1")
}

func TestDatabaseService_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ provider_domain.ResourceDescriptor = (*databaseService)(nil)
	var _ provider_domain.ResourceTypeDescriptor = (*databaseService)(nil)
}

func assertInfoEntry(t *testing.T, entries []provider_domain.InfoEntry, key, expectedValue string) {
	t.Helper()
	for _, e := range entries {
		if e.Key == key {
			assert.Equal(t, expectedValue, e.Value, "entry %q", key)
			return
		}
	}
	t.Errorf("expected entry with key %q not found", key)
}

type mockHealthChecker struct {
	diagnostics []DatabaseHealthDiagnostic
}

func (m mockHealthChecker) CheckHealth(_ context.Context, _ *sql.DB) []DatabaseHealthDiagnostic {
	return m.diagnostics
}
