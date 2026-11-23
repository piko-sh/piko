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
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	"piko.sh/piko/internal/wal/wal_domain"
)

func TestGeneratedPikoScenario(t *testing.T) {
	if dir := os.Getenv("PIKO_FUZZ_UNCLEAN_WAL"); dir != "" {
		writeAndExit(t, dir)
		return
	}
	token := os.Getenv("PIKO_FUZZ_CASE_TOKEN")
	draws, err := hex.DecodeString(token)
	require.NoError(t, err)
	require.Len(t, draws, 8)
	ctx := t.Context()
	cfg := Config{RegistryCapacity: 16, OrchestratorCapacity: 32, Persistence: &PersistenceProviderConfig{
		Enabled: true, WALDir: t.TempDir(), SyncMode: wal_domain.SyncModeEveryWrite, SnapshotThreshold: 100,
	}}
	open := func() (*Provider, *otter.DAL) {
		t.Helper()
		p := NewProvider(cfg)
		require.NoError(t, p.Connect(ctx))
		factory, err := p.OrchestratorDALFactory()
		require.NoError(t, err)
		store, err := factory.NewOrchestratorDAL()
		require.NoError(t, err)
		dal, ok := store.(*otter.DAL)
		require.True(t, ok)
		return p, dal
	}
	p, store := open()
	defer func() { require.NoError(t, p.Close(ctx)) }()
	raw, err := hex.DecodeString(os.Getenv("PIKO_FUZZ_DRAWS"))
	require.NoError(t, err)
	ids := []string{token + "-a", token + "-b", token + "-c", token + "-d"}
	type record struct {
		attempt int
		message string
	}
	want := map[string]record{}
	steps := append([]byte{0, 5, 2, 1, 3, 6, 10, 4}, raw...)
	steps = append(steps, 3)
	for i, b := range steps {
		id := ids[int(b/5)%len(ids)]
		switch b % 5 {
		case 0, 1:
			_, exists := want[id]
			r := record{attempt: int(b) + i, message: fmt.Sprintf("event-%d-%d", i, b)}
			task := fullTask(id)
			task.Attempt, task.LastError = r.attempt, r.message
			if exists {
				err = store.UpdateTask(ctx, task)
			} else {
				err = store.CreateTask(ctx, task)
			}
			require.NoError(t, err)
			want[id] = r
		case 2:
			require.NoError(t, p.Checkpoint(ctx))
		case 3:
			require.NoError(t, p.Close(ctx))
			p, store = open()
		case 4:
		}
		found, err := store.GetTasksByID(ctx, ids)
		require.NoError(t, err)
		got := map[string]record{}
		for _, r := range found {
			got[r.ID] = record{attempt: r.Attempt, message: r.LastError}
			expected := fullTask(r.ID)
			expected.Attempt, expected.LastError = want[r.ID].attempt, want[r.ID].message
			assertSameTask(t, expected, r)
		}
		require.Equal(t, want, got, "PIKO_CHECK task-history: step %d byte %d", i, b)
	}

	t.Run("unclean-reopen", func(t *testing.T) {
		dir := t.TempDir()
		attempt := 1 + int(draws[0])%8
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGeneratedPikoScenario$")
		cmd.Env = append(os.Environ(), "PIKO_FUZZ_UNCLEAN_WAL="+dir, "PIKO_FUZZ_UNCLEAN_ATTEMPT="+strconv.Itoa(attempt))
		log, err := cmd.CombinedOutput()
		exit, ok := errors.AsType[*exec.ExitError](err)
		require.True(t, ok && exit.ExitCode() == 38, "PIKO_CHECK interrupted-stage: child did not reach synced update: %s", log)
		prior, db := openCrashStore(t, dir)
		defer func() { require.NoError(t, prior.Close(ctx)) }()
		rows, err := db.GetTasksByID(ctx, []string{"persisted", "unstarted"})
		require.NoError(t, err)
		require.Len(t, rows, 1, "PIKO_CHECK crash-recovery: unexpected committed rows")
		require.Equal(t, "persisted", rows[0].ID)
		require.Equal(t, attempt, rows[0].Attempt, "PIKO_CHECK crash-tail: saved update was lost")
		require.Equal(t, "acknowledged", rows[0].LastError)
		rows[0].Attempt++
		require.NoError(t, db.UpdateTask(ctx, rows[0]))
		again, err := db.GetTasksByID(ctx, []string{"persisted"})
		require.NoError(t, err)
		require.Len(t, again, 1)
		require.Equal(t, attempt+1, again[0].Attempt, "PIKO_CHECK crash-continue: failed to advance after reopening")
	})
}
func openCrashStore(t *testing.T, dir string) (*Provider, *otter.DAL) {
	t.Helper()
	p := NewProvider(Config{RegistryCapacity: 16, OrchestratorCapacity: 32, Persistence: &PersistenceProviderConfig{Enabled: true, WALDir: dir, SyncMode: wal_domain.SyncModeEveryWrite, SnapshotThreshold: 100}})
	require.NoError(t, p.Connect(t.Context()))
	factory, err := p.OrchestratorDALFactory()
	require.NoError(t, err)
	store, err := factory.NewOrchestratorDAL()
	require.NoError(t, err)
	return p, store.(*otter.DAL)
}
func writeAndExit(t *testing.T, dir string) {
	t.Helper()
	p, store := openCrashStore(t, dir)
	task := fullTask("persisted")
	require.NoError(t, store.CreateTask(t.Context(), task))
	require.NoError(t, p.Checkpoint(t.Context()))
	attempt, err := strconv.Atoi(os.Getenv("PIKO_FUZZ_UNCLEAN_ATTEMPT"))
	require.NoError(t, err)
	task.Attempt, task.LastError = attempt, "acknowledged"
	require.NoError(t, store.UpdateTask(t.Context(), task))
	os.Exit(38)
}
