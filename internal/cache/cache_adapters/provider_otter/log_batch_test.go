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

package provider_otter

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/wal/wal_adapters/driven_disk"
	"piko.sh/piko/internal/wal/wal_domain"
)

func openArticleLog(t *testing.T, dir string) *OtterAdapter[string, testArticle] {
	t.Helper()
	walConfig := wal_domain.DefaultConfig(dir)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = 1 << 30
	provider, err := OtterProviderFactory(cache_dto.Options[string, testArticle]{
		MaximumEntries: 1000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	})
	require.NoError(t, err, "the logged cache must open")
	adapter, ok := provider.(*OtterAdapter[string, testArticle])
	require.True(t, ok, "the factory must return the otter adapter")
	return adapter
}

func crashCopy(t *testing.T, dir string) string {
	t.Helper()
	copyDir := t.TempDir()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		require.NoError(t, readErr)
		require.NoError(t, os.WriteFile(filepath.Join(copyDir, entry.Name()), data, 0o600))
	}
	return copyDir
}

func logRecords(t *testing.T, dir string) []wal_domain.RawEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, wal_domain.DefaultConfig(dir).WALFileName))
	require.NoError(t, err)
	records, err := driven_disk.DecodeRawEntries(data)
	require.NoError(t, err)
	return records
}

func article(title string) testArticle { return testArticle{Title: title} }

func present(adapter *OtterAdapter[string, testArticle], key string) bool {
	_, ok, _ := adapter.GetIfPresent(context.Background(), key)
	return ok
}

func TestLogBatchTransactionIsOneRecord(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	adapter := openArticleLog(t, dir)
	defer func() { _ = adapter.Close(ctx) }()

	require.NoError(t, adapter.Set(ctx, "before", article("before")))
	transaction := cache_domain.BeginTransaction(ctx, adapter)
	require.NoError(t, transaction.Set(ctx, "first", article("first")))
	require.NoError(t, transaction.Set(ctx, "second", article("second")))
	require.NoError(t, transaction.Invalidate(ctx, "before"))
	require.NoError(t, transaction.Commit(ctx))

	records := logRecords(t, dir)
	require.Len(t, records, 2, "the transaction's three writes must reach the log as one record")
	assert.Equal(t, wal_domain.OpBatch, records[1].Operation)

	recovered := openArticleLog(t, crashCopy(t, dir))
	defer func() { _ = recovered.Close(ctx) }()
	assert.True(t, present(recovered, "first"), "a committed transaction replays")
	assert.True(t, present(recovered, "second"), "a committed transaction replays")
	assert.False(t, present(recovered, "before"), "a committed removal replays")
}

func TestLogBatchTornTransactionIsLostWhole(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	adapter := openArticleLog(t, dir)
	defer func() { _ = adapter.Close(ctx) }()

	require.NoError(t, adapter.Set(ctx, "before", article("before")))
	transaction := cache_domain.BeginTransaction(ctx, adapter)
	require.NoError(t, transaction.Set(ctx, "first", article("first")))
	require.NoError(t, transaction.Set(ctx, "second", article("second")))
	require.NoError(t, transaction.Commit(ctx))

	crashed := crashCopy(t, dir)
	logPath := filepath.Join(crashed, wal_domain.DefaultConfig(crashed).WALFileName)
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	batchStart := 4 + int(binary.BigEndian.Uint32(data))
	batchEnd := batchStart + 4 + int(binary.BigEndian.Uint32(data[batchStart:]))
	require.NoError(t, os.Truncate(logPath, int64(batchEnd-5)), "tear the transaction's record as a crash mid-write would")

	recovered := openArticleLog(t, crashed)
	defer func() { _ = recovered.Close(ctx) }()
	assert.True(t, present(recovered, "before"), "records before the torn one survive")
	assert.False(t, present(recovered, "first"), "no part of a torn transaction survives")
	assert.False(t, present(recovered, "second"), "no part of a torn transaction survives")
}

func TestLogBatchRollbackNeverReachesLog(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	adapter := openArticleLog(t, dir)
	defer func() { _ = adapter.Close(ctx) }()

	require.NoError(t, adapter.Set(ctx, "kept", article("kept")))
	transaction := cache_domain.BeginTransaction(ctx, adapter)
	require.NoError(t, transaction.Set(ctx, "kept", article("overwritten")))
	require.NoError(t, transaction.Set(ctx, "added", article("added")))
	require.NoError(t, transaction.Rollback(ctx))

	assert.Len(t, logRecords(t, dir), 1, "neither a rolled-back transaction nor its undo reaches the log")
	kept, ok, err := adapter.GetIfPresent(ctx, "kept")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "kept", kept.Title, "rollback restores the cache")
	assert.False(t, present(adapter, "added"), "rollback removes what the transaction added")

	require.NoError(t, adapter.Set(ctx, "after", article("after")))
	assert.Len(t, logRecords(t, dir), 2, "writes after a rollback log normally")
}

func TestLogBatchFailedLogWriteLeavesCacheUnchanged(t *testing.T) {
	ctx := context.Background()
	adapter := openArticleLog(t, t.TempDir())
	defer func() { _ = adapter.Close(ctx) }()

	require.NoError(t, adapter.wal.Close(), "close the log underneath the cache")

	assert.Error(t, adapter.Set(ctx, "lost", article("lost")), "a write the log refuses must fail")
	assert.False(t, present(adapter, "lost"), "a write the log refused must not reach the cache")

	transaction := cache_domain.BeginTransaction(ctx, adapter)
	require.NoError(t, transaction.Set(ctx, "pending", article("pending")))
	require.Error(t, transaction.Commit(ctx), "a transaction the log refuses must fail to commit")
	require.NoError(t, transaction.Rollback(ctx), "the failed commit leaves the transaction to roll back")
	assert.False(t, present(adapter, "pending"), "rolling back a refused transaction restores the cache")
}
