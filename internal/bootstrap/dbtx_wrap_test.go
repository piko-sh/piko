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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	registrydb "piko.sh/piko/internal/registry/registry_dal/querier_sqlite/db"
)

func TestWrapDBTX_CarriesTheInstrumentationOntoAnotherConnection(t *testing.T) {
	t.Parallel()

	observer := &captureObserver{}
	resolver := func(string) string { return "RESOLVED" }
	original := newOTelDBTX(&stubDBTX{}, "sqlite", "testdb", resolver, observer)

	other := &stubDBTX{}
	wrapped, ok := original.WrapDBTX(other).(*otelDBTX)
	require.True(t, ok)

	assert.Same(t, other, wrapped.inner, "the clone instruments the NEW connection")
	assert.NotSame(t, original, wrapped, "the original keeps its own inner")
	assert.Equal(t, original.databaseSystem, wrapped.databaseSystem)
	assert.Equal(t, original.databaseNamespace, wrapped.databaseNamespace)
	assert.Equal(t, original.tracer, wrapped.tracer)
	assert.Equal(t, original.observer, wrapped.observer)
	require.NotNil(t, wrapped.resolver)
	assert.Equal(t, "RESOLVED", wrapped.resolver("anything"))
}

func TestWrapDBTX_ANonConnectionIsReturnedUnchanged(t *testing.T) {
	t.Parallel()

	original := newOTelDBTX(&stubDBTX{}, "sqlite", "testdb", nil, nil)

	assert.Equal(t, "not a connection", original.WrapDBTX("not a connection"))
}

func TestGeneratedQuerier_WithTxStaysInstrumented(t *testing.T) {
	t.Parallel()

	observer := &captureObserver{}
	inner := &stubDBTX{}
	instrumented := newOTelDBTX(inner, "sqlite", "testdb", nil, observer)

	database, err := fakeDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	transaction, err := database.Begin()
	require.NoError(t, err)
	t.Cleanup(func() { _ = transaction.Rollback() })

	queries := registrydb.New(instrumented)
	inTx := queries.WithTx(transaction)

	require.NoError(t, inTx.AddGCHint(context.Background(), registrydb.AddGCHintParams{
		BackendID: "b-1", StorageKey: "k-1", CreatedAt: 1,
	}))

	require.Len(t, observer.obs, 1, "a statement run inside a transaction must still be observed")
	assert.Equal(t, "testdb", observer.obs[0].Connection)
	assert.Equal(t, "sqlite", observer.obs[0].System)
}

func TestGeneratedQuerier_AnUninstrumentedQuerierIsUnaffected(t *testing.T) {
	t.Parallel()

	database, err := fakeDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	transaction, err := database.Begin()
	require.NoError(t, err)
	t.Cleanup(func() { _ = transaction.Rollback() })

	queries := registrydb.New(&stubDBTX{})

	assert.NotPanics(t, func() { _ = queries.WithTx(transaction) })
}

func TestGeneratedQuerier_WithTxOnANilReceiver(t *testing.T) {
	t.Parallel()

	database, err := fakeDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	transaction, err := database.Begin()
	require.NoError(t, err)
	t.Cleanup(func() { _ = transaction.Rollback() })

	var queries *registrydb.Queries

	require.NotPanics(t, func() {
		inTx := queries.WithTx(transaction)
		require.NotNil(t, inTx)
		require.NoError(t, inTx.AddGCHint(context.Background(), registrydb.AddGCHintParams{
			BackendID: "b-1", StorageKey: "k-1", CreatedAt: 1,
		}))
	})
}

func TestWrapDBTX_UnusableWrapperStillYieldsAWorkingConnection(t *testing.T) {
	t.Parallel()

	observer := &captureObserver{}
	instrumented := newOTelDBTX(&stubDBTX{}, "sqlite", "testdb", nil, observer)

	got := instrumented.WrapDBTX(struct{ name string }{name: "not a connection"})

	assert.NotNil(t, got, "the caller always receives something")
	assert.IsType(t, struct{ name string }{}, got, "an unusable value comes back unchanged")
}
