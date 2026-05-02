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

package migration_sql

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresAdvisoryLock_RejectsInvalidLockKey(t *testing.T) {
	t.Parallel()

	lock := &PostgresAdvisoryLock{LockKey: "drop table users; --"}

	_, err := lock.key()

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidIdentifier))
	assert.Contains(t, err.Error(), "PostgresAdvisoryLock.LockKey")
}

func TestMySQLAdvisoryLock_RejectsInvalidLockKey(t *testing.T) {
	t.Parallel()

	lock := &MySQLAdvisoryLock{LockKey: "drop table users; --"}

	_, err := lock.key()

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidIdentifier))
	assert.Contains(t, err.Error(), "MySQLAdvisoryLock.LockKey")
}

func TestPostgresAdvisoryLock_AcceptsValidLockKey(t *testing.T) {
	t.Parallel()

	lock := &PostgresAdvisoryLock{LockKey: "tenant_one"}

	key, err := lock.key()

	require.NoError(t, err)
	assert.Equal(t, "tenant_one", key)
}

func TestMySQLAdvisoryLock_AcceptsValidLockKey(t *testing.T) {
	t.Parallel()

	lock := &MySQLAdvisoryLock{LockKey: "tenant_one"}

	key, err := lock.key()

	require.NoError(t, err)
	assert.Equal(t, "tenant_one", key)
}

func TestPostgresAdvisoryLock_FallsBackToDefaultWhenKeyEmpty(t *testing.T) {
	t.Parallel()

	lock := &PostgresAdvisoryLock{}

	key, err := lock.key()

	require.NoError(t, err)
	assert.Equal(t, DefaultHistoryTableName, key)
}

func TestMySQLAdvisoryLock_FallsBackToDefaultWhenKeyEmpty(t *testing.T) {
	t.Parallel()

	lock := &MySQLAdvisoryLock{}

	key, err := lock.key()

	require.NoError(t, err)
	assert.Equal(t, DefaultHistoryTableName, key)
}
