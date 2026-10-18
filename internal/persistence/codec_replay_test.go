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

package persistence

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cache_adapters_otter "piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	d "piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/wal/wal_domain"
)

type legacyTaskCodec struct{}

func (legacyTaskCodec) EncodeValue(v *d.Task) ([]byte, error) { return json.Marshal(v) }

func (legacyTaskCodec) DecodeValue(data []byte) (*d.Task, error) {
	var v d.Task
	return &v, json.Unmarshal(data, &v)
}

func fullTask(id string) *d.Task {
	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	return &d.Task{
		ID: id, WorkflowID: "wf-" + id, Executor: "artefact.compiler", Status: d.StatusRetrying,
		LastError: "temporary", DeduplicationKey: "dedup:" + id, BuildTag: "build-9", OwnerInstance: "node-a",
		Attempt: 3, IsFatal: true, Background: true,
		ExecuteAt: at, CreatedAt: at.Add(-time.Hour), UpdatedAt: at.Add(-time.Minute),
		ScheduledExecuteAt: at.Add(time.Hour), LastPublishedAt: at.Add(-time.Second),
		Payload: map[string]any{"artefactID": "assets/" + id + ".svg", "widths": []any{float64(320), float64(640)}},
		Result:  map[string]any{"bytes": float64(2048)},
		Config:  d.TaskConfig{Timeout: 45 * time.Second, MaxRetries: 4, Priority: d.PriorityLow},
		Rerun: &d.RerunRequest{
			ID: id + "-rerun", Payload: map[string]any{"version": float64(2)},
			Config: d.TaskConfig{Timeout: time.Minute, MaxRetries: 2, Priority: d.PriorityHigh},
		},
	}
}

func assertSameTask(t *testing.T, want, got *d.Task) {
	t.Helper()
	for _, pair := range [][2]time.Time{
		{want.ExecuteAt, got.ExecuteAt}, {want.CreatedAt, got.CreatedAt}, {want.UpdatedAt, got.UpdatedAt},
		{want.ScheduledExecuteAt, got.ScheduledExecuteAt}, {want.LastPublishedAt, got.LastPublishedAt},
	} {
		assert.True(t, pair[0].Equal(pair[1]), "time %s, want %s", pair[1], pair[0])
	}
	wantCopy, gotCopy := *want, *got
	for _, task := range []*d.Task{&wantCopy, &gotCopy} {
		task.ExecuteAt, task.CreatedAt, task.UpdatedAt = time.Time{}, time.Time{}, time.Time{}
		task.ScheduledExecuteAt, task.LastPublishedAt = time.Time{}, time.Time{}
	}
	assert.Equal(t, wantCopy, gotCopy)
}

func TestTaskCodec_RoundTripsEveryField(t *testing.T) {
	codec := TaskCodec{}
	for _, task := range []*d.Task{
		fullTask("full"),
		{ID: "bare", Status: d.StatusPending},
		{ID: "empty-maps", Payload: map[string]any{}, Result: map[string]any{}, Rerun: &d.RerunRequest{ID: "r"}},
	} {
		encoded, err := codec.EncodeValue(task)
		require.NoError(t, err)
		require.Equal(t, taskCodecVersion2, encoded[0])
		decoded, err := codec.DecodeValue(encoded)
		require.NoError(t, err)
		assertSameTask(t, task, decoded)
	}
}

func TestTaskCodec_DecodesLegacyJSON(t *testing.T) {
	task := fullTask("legacy")
	encoded, err := json.Marshal(task)
	require.NoError(t, err)
	decoded, err := TaskCodec{}.DecodeValue(encoded)
	require.NoError(t, err)
	assertSameTask(t, task, decoded)
}

func TestTaskCodec_RejectsTruncatedEncoding(t *testing.T) {
	encoded, err := TaskCodec{}.EncodeValue(fullTask("cut"))
	require.NoError(t, err)
	for _, cut := range []int{2, len(encoded) / 2, len(encoded) - 1} {
		_, err := TaskCodec{}.DecodeValue(encoded[:cut])
		assert.Error(t, err, "an encoding cut at %d bytes must not decode", cut)
	}
}

func TestTaskCodec_EncodingIsSmallerThanJSON(t *testing.T) {
	task := fullTask("size")
	binaryForm, err := TaskCodec{}.EncodeValue(task)
	require.NoError(t, err)
	jsonForm, err := json.Marshal(task)
	require.NoError(t, err)
	assert.Less(t, len(binaryForm), len(jsonForm))
}

func TestOrchestratorWALReplaysLegacyAndBinaryTasks(t *testing.T) {
	walDir := filepath.Join(t.TempDir(), "wal")
	ctx := context.Background()

	legacyOptions := newOtterCacheOptions[string, *d.Task](0, cache_adapters_otter.PersistenceConfig[string, *d.Task]{
		Enabled:    true,
		WALConfig:  buildWALConfig(walDir, "orchestrator", wal_domain.SyncModeEveryWrite, 1000),
		KeyCodec:   StringKeyCodec{},
		ValueCodec: legacyTaskCodec{},
	})
	legacyCache, err := cache_adapters_otter.OtterProviderFactory(legacyOptions)
	require.NoError(t, err)
	legacy := fullTask("written-as-json")
	require.NoError(t, legacyCache.Set(ctx, legacy.ID, legacy))
	require.NoError(t, legacyCache.Close(ctx))

	cfg := Config{RegistryCapacity: 10, OrchestratorCapacity: 100, Persistence: &PersistenceProviderConfig{
		Enabled: true, WALDir: walDir, SyncMode: wal_domain.SyncModeEveryWrite, SnapshotThreshold: 1000,
	}}
	open := func() (*Provider, *otter.DAL) {
		p := NewProvider(cfg)
		require.NoError(t, p.Connect(ctx))
		factory, err := p.OrchestratorDALFactory()
		require.NoError(t, err)
		raw, err := factory.NewOrchestratorDAL()
		require.NoError(t, err)
		store, ok := raw.(*otter.DAL)
		require.True(t, ok)
		return p, store
	}

	p, store := open()
	rows, err := store.GetTasksByID(ctx, []string{legacy.ID})
	require.NoError(t, err)
	require.Len(t, rows, 1, "a task written as JSON replays")
	assertSameTask(t, legacy, rows[0])
	binary := fullTask("written-as-binary")
	binary.DeduplicationKey = "dedup:binary"
	require.NoError(t, store.CreateTask(ctx, binary))
	require.NoError(t, p.Close(ctx))

	p, store = open()
	defer func() { require.NoError(t, p.Close(ctx)) }()
	rows, err = store.GetTasksByID(ctx, []string{legacy.ID, binary.ID})
	require.NoError(t, err)
	require.Len(t, rows, 2, "both encodings replay from one log")
}
