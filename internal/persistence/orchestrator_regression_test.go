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
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	d "piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/wal/wal_domain"
)

func TestOrchestratorWALRepairsLegacyDatesAndPersistsTransitions(t *testing.T) {
	cfg := Config{RegistryCapacity: 10, OrchestratorCapacity: 4, Persistence: &PersistenceProviderConfig{
		Enabled: true, WALDir: filepath.Join(t.TempDir(), "wal"), SyncMode: wal_domain.SyncModeEveryWrite, SnapshotThreshold: 1000,
	}}
	open := func() (*Provider, *otter.DAL) {
		p := NewProvider(cfg)
		require.NoError(t, p.Connect(t.Context()))
		f, err := p.OrchestratorDALFactory()
		require.NoError(t, err)
		raw, err := f.NewOrchestratorDAL()
		require.NoError(t, err)
		store, ok := raw.(*otter.DAL)
		require.True(t, ok)
		return p, store
	}
	p, store := open()
	future := d.NewTask("test", nil)
	future.Status = d.StatusScheduled
	future.ExecuteAt = time.Now().Add(time.Hour)
	future.ScheduledExecuteAt = time.Time{}
	require.NoError(t, store.CreateTask(t.Context(), future))
	due := d.NewTask("test", nil)
	due.Status = d.StatusScheduled
	due.ExecuteAt = time.Now().Add(-time.Minute)
	due.ScheduledExecuteAt = due.ExecuteAt
	require.NoError(t, store.CreateTask(t.Context(), due))
	processing := d.NewTask("test", nil)
	processing.Status = d.StatusProcessing
	processing.UpdatedAt = time.Now().Add(-time.Hour)
	require.NoError(t, store.CreateTask(t.Context(), processing))
	stale := d.NewTask("test", nil)
	stale.Status = d.StatusProcessing
	stale.UpdatedAt = time.Now().Add(-time.Hour)
	require.NoError(t, store.CreateTask(t.Context(), stale))
	require.ErrorIs(t, store.CreateTask(t.Context(), d.NewTask("overflow", nil)), d.ErrTaskCapacity)
	require.NoError(t, p.Close(t.Context()))

	p, store = open()
	rows, err := store.GetTasksByID(t.Context(), []string{future.ID})
	require.NoError(t, err)
	require.True(t, rows[0].ScheduledExecuteAt.Equal(future.ExecuteAt.Truncate(time.Second)), "times are stored at whole seconds")
	n, err := store.PromoteScheduledTasks(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	beating := *processing
	beating.UpdatedAt = time.Now()
	_, err = store.TransitionTask(t.Context(), d.Transition{
		Task: &beating, From: []d.TaskStatus{d.StatusProcessing},
	})
	require.NoError(t, err, "a refreshed processing task stays processing")
	orphans, err := store.ListOrphanedTasks(t.Context(), d.OrphanQuery{
		Now: time.Now(), StaleBefore: time.Now().Add(-time.Minute), Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, orphans, 1, "only the task idle past the cutoff is an orphan")
	require.Equal(t, stale.ID, orphans[0].ID)
	recovered := orphans[0]
	recovered.Status = d.StatusRetrying
	recovered.LastError = "recovered"
	recovered.UpdatedAt = time.Now()
	_, err = store.TransitionTask(t.Context(), d.Transition{
		Task: recovered, From: []d.TaskStatus{d.StatusProcessing},
	})
	require.NoError(t, err)
	require.NoError(t, p.Close(t.Context()))

	p, store = open()
	defer p.Close(t.Context())
	rows, err = store.GetTasksByID(t.Context(), []string{future.ID, due.ID, processing.ID, stale.ID})
	require.NoError(t, err)
	require.Len(t, rows, 4)
	require.Equal(t, d.StatusScheduled, rows[0].Status)
	require.True(t, rows[0].ScheduledExecuteAt.Equal(future.ExecuteAt.Truncate(time.Second)), "times are stored at whole seconds")
	require.Equal(t, d.StatusPending, rows[1].Status)
	require.Equal(t, d.StatusProcessing, rows[2].Status)
	require.WithinDuration(t, time.Now(), rows[2].UpdatedAt, 10*time.Second)
	require.Equal(t, d.StatusRetrying, rows[3].Status)
	require.Equal(t, 0, rows[3].Attempt, "recovery keeps the attempt; the next claim counts the next one")
	require.Equal(t, "recovered", rows[3].LastError)
}
