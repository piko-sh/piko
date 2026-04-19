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

package migration_sql_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
)

func TestNoOpLock_AcquireReturnsNilConnectionAndError(t *testing.T) {
	t.Parallel()

	lock := &migration_sql.NoOpLock{}

	connection, err := lock.Acquire(context.Background(), nil)

	require.NoError(t, err)
	require.Nil(t, connection)
}

func TestNoOpLock_TryAcquireReturnsNilConnectionAndError(t *testing.T) {
	t.Parallel()

	lock := &migration_sql.NoOpLock{}

	connection, err := lock.TryAcquire(context.Background(), nil)

	require.NoError(t, err)
	require.Nil(t, connection)
}

func TestNoOpLock_ReleaseAcceptsNilConnection(t *testing.T) {
	t.Parallel()

	lock := &migration_sql.NoOpLock{}

	require.NoError(t, lock.Release(context.Background(), nil))
}

func TestPostgresAdvisoryLock_ImplementsLockStrategy(t *testing.T) {
	t.Parallel()

	var _ migration_sql.LockStrategy = (*migration_sql.PostgresAdvisoryLock)(nil)
}

func TestMySQLAdvisoryLock_ImplementsLockStrategy(t *testing.T) {
	t.Parallel()

	var _ migration_sql.LockStrategy = (*migration_sql.MySQLAdvisoryLock)(nil)
}

func TestTableBasedLock_ImplementsLockStrategy(t *testing.T) {
	t.Parallel()

	var _ migration_sql.LockStrategy = (*migration_sql.TableBasedLock)(nil)
}

func TestPostgresAdvisoryLock_ReleaseAcceptsNilConnection(t *testing.T) {
	t.Parallel()

	lock := &migration_sql.PostgresAdvisoryLock{}

	require.NoError(t, lock.Release(context.Background(), nil))
}

func TestMySQLAdvisoryLock_ReleaseAcceptsNilConnection(t *testing.T) {
	t.Parallel()

	lock := &migration_sql.MySQLAdvisoryLock{}

	require.NoError(t, lock.Release(context.Background(), nil))
}

func TestTableBasedLock_ReleaseAcceptsNilConnection(t *testing.T) {
	t.Parallel()

	lock := &migration_sql.TableBasedLock{
		CreateLockTableSQL: "CREATE TABLE",
	}

	require.NoError(t, lock.Release(context.Background(), nil))
}
