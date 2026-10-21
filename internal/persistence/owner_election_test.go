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
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_dal"
)

const (
	ownerHelperDirEnv = "PIKO_OWNER_ELECTION_HELPER_DIR"
)

type taskQueue interface {
	CreateTask(ctx context.Context, task *orchestrator_domain.Task) error
	PendingTaskCount(ctx context.Context) (int64, error)
}

func registryOf(t *testing.T, provider *Provider) registry_dal.RegistryDAL {
	t.Helper()
	factory, err := provider.RegistryDALFactory()
	require.NoError(t, err, "the registry DAL factory must be available")
	registryDAL, err := factory.NewRegistryDAL()
	require.NoError(t, err, "a registry DAL must be constructable")
	dal, ok := registryDAL.(registry_dal.RegistryDAL)
	require.True(t, ok, "the registry DAL must satisfy registry_dal.RegistryDAL")
	return dal
}

func queueOf(t *testing.T, provider *Provider) taskQueue {
	t.Helper()
	factory, err := provider.OrchestratorDALFactory()
	require.NoError(t, err, "the orchestrator DAL factory must be available")
	orchestratorDAL, err := factory.NewOrchestratorDAL()
	require.NoError(t, err, "an orchestrator DAL must be constructable")
	queue, ok := orchestratorDAL.(taskQueue)
	require.True(t, ok, "the orchestrator DAL must create and count tasks")
	return queue
}

func hasArtefact(ctx context.Context, provider *Provider, t *testing.T, id string) bool {
	t.Helper()
	_, err := registryOf(t, provider).GetArtefact(ctx, id)
	return err == nil
}

func directoryContents(t *testing.T, dir string) map[string]string {
	t.Helper()
	contents := make(map[string]string)
	require.NoError(t, filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		relative, _ := filepath.Rel(dir, path)
		contents[relative] = string(data)
		return nil
	}), "the persistence directory must be readable")
	return contents
}

func TestOwnerElectionSecondInstanceRunsFromReadOnlyCopy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	walDir := filepath.Join(t.TempDir(), "wal")

	owner := persistenceProvider(t, walDir)
	require.NoError(t, owner.Connect(ctx), "the first instance must connect")
	assert.False(t, owner.IsSecondary(), "the first instance must own the directory")
	writeCheckpointArtefact(t, owner, ctx, "art-owner")
	require.NoError(t, queueOf(t, owner).CreateTask(ctx, orchestrator_domain.NewTask("owner.work", nil)),
		"the owner must queue a task")

	before := directoryContents(t, walDir)

	secondary := persistenceProvider(t, walDir)
	require.NoError(t, secondary.Connect(ctx), "a second instance must never fail to start")
	assert.True(t, secondary.IsSecondary(), "the second instance must run as a secondary")
	assert.True(t, hasArtefact(ctx, secondary, t, "art-owner"),
		"the secondary must start warm with the owner's registry")

	pending, err := queueOf(t, secondary).PendingTaskCount(ctx)
	require.NoError(t, err, "the secondary's queue must be countable")
	assert.Zero(t, pending, "the secondary must not take on the owner's queued work")

	writeCheckpointArtefact(t, secondary, ctx, "art-secondary")
	require.NoError(t, queueOf(t, secondary).CreateTask(ctx, orchestrator_domain.NewTask("secondary.work", nil)),
		"the secondary must queue tasks in memory")
	require.NoError(t, secondary.Checkpoint(ctx), "a secondary checkpoint must be a no-op")
	require.NoError(t, secondary.Close(ctx), "the secondary must close")

	assert.Equal(t, before, directoryContents(t, walDir),
		"the secondary must leave the owner's directory byte-identical")
	assert.False(t, hasArtefact(ctx, owner, t, "art-secondary"),
		"the secondary's writes must stay in its own memory")

	require.NoError(t, owner.Close(ctx), "the owner must close")

	successor := persistenceProvider(t, walDir)
	require.NoError(t, successor.Connect(ctx), "the next instance must connect")
	defer func() { _ = successor.Close(ctx) }()
	assert.False(t, successor.IsSecondary(), "once the owner exits the next instance must take ownership")
	assert.True(t, hasArtefact(ctx, successor, t, "art-owner"), "the owner's registry must survive")
	assert.False(t, hasArtefact(ctx, successor, t, "art-secondary"),
		"nothing the secondary wrote may reach the directory")
}

func TestOwnerElectionSecondaryStagesRegistrySnapshot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	walDir := filepath.Join(t.TempDir(), "wal")

	owner := persistenceProvider(t, walDir)
	require.NoError(t, owner.Connect(ctx), "the owner must connect")
	defer func() { _ = owner.Close(ctx) }()
	writeCheckpointArtefact(t, owner, ctx, "art-owner")

	secondary := persistenceProvider(t, walDir)
	require.NoError(t, secondary.Connect(ctx), "the secondary must connect")
	defer func() { _ = secondary.Close(ctx) }()
	writeCheckpointArtefact(t, secondary, ctx, "art-secondary")

	staging := filepath.Join(t.TempDir(), "staging")
	require.NoError(t, secondary.WriteRegistrySnapshot(ctx, staging),
		"the secondary must stage its registry outside the owner's directory")

	reader := persistenceProvider(t, staging)
	require.NoError(t, reader.Connect(ctx), "the staged snapshot must load")
	defer func() { _ = reader.Close(ctx) }()
	assert.True(t, hasArtefact(ctx, reader, t, "art-owner"), "the staged snapshot must hold the warm copy")
	assert.True(t, hasArtefact(ctx, reader, t, "art-secondary"), "the staged snapshot must hold the secondary's work")
}

func TestOwnerElectionAcrossProcesses(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("re-executes the test binary")
	}

	ctx := context.Background()
	walDir := filepath.Join(t.TempDir(), "wal")

	owner := persistenceProvider(t, walDir)
	require.NoError(t, owner.Connect(ctx), "the owner must connect")
	defer func() { _ = owner.Close(ctx) }()
	writeCheckpointArtefact(t, owner, ctx, "art-owner")
	before := directoryContents(t, walDir)

	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestOwnerElectionHelperProcess$", "-test.count=1")
	command.Env = append(os.Environ(), ownerHelperDirEnv+"="+walDir)
	output, err := command.CombinedOutput()
	require.NoError(t, err, "the second process must start and finish cleanly:\n%s", output)
	assert.Contains(t, string(output), "helper: secondary warm", "the second process must run warm as a secondary:\n%s", output)
	assert.Contains(t, string(output), "owner_pid="+strconv.Itoa(os.Getpid()),
		"the second process must name the owner's pid:\n%s", output)

	assert.Equal(t, before, directoryContents(t, walDir),
		"the second process must leave the owner's directory byte-identical")
}

func TestOwnerElectionHelperProcess(t *testing.T) {
	walDir := os.Getenv(ownerHelperDirEnv)
	if walDir == "" {
		t.Skip("only runs as TestOwnerElectionAcrossProcesses' second process")
	}

	ctx := context.Background()
	secondary := persistenceProvider(t, walDir)
	require.NoError(t, secondary.Connect(ctx), "the second process must connect")
	require.True(t, secondary.IsSecondary(), "the second process must run as a secondary")
	require.True(t, hasArtefact(ctx, secondary, t, "art-owner"), "the second process must start warm")
	writeCheckpointArtefact(t, secondary, ctx, "art-helper")
	require.NoError(t, secondary.Checkpoint(ctx), "the secondary checkpoint must be a no-op")
	require.NoError(t, secondary.Close(ctx), "the second process must close")

	var report strings.Builder
	report.WriteString("helper: secondary warm owner_pid=")
	report.WriteString(strconv.Itoa(secondary.ownerPID))
	t.Log(report.String())
	_, _ = os.Stdout.WriteString(report.String() + "\n")
}

func TestCorruptOrchestratorStateIsQuarantined(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	walDir := filepath.Join(t.TempDir(), "wal")

	first := persistenceProvider(t, walDir)
	require.NoError(t, first.Connect(ctx))
	writeCheckpointArtefact(t, first, ctx, "art-kept")
	require.NoError(t, queueOf(t, first).CreateTask(ctx, orchestrator_domain.NewTask("queued.work", nil)))
	require.NoError(t, first.Close(ctx))

	snapshotPath := filepath.Join(walDir, "orchestrator", "snapshot.piko")
	require.FileExists(t, snapshotPath, "closing must have checkpointed the queue")
	require.NoError(t, os.WriteFile(snapshotPath, []byte("written by something else entirely"), 0o600))

	second := persistenceProvider(t, walDir)
	require.NoError(t, second.Connect(ctx), "an unreadable work queue must not stop the process starting")
	defer func() { _ = second.Close(ctx) }()

	pending, err := queueOf(t, second).PendingTaskCount(ctx)
	require.NoError(t, err)
	assert.Zero(t, pending, "the queue starts empty once its unreadable state is moved aside")
	assert.True(t, hasArtefact(ctx, second, t, "art-kept"), "the registry is untouched")

	quarantined, err := filepath.Glob(snapshotPath + ".corrupt-*")
	require.NoError(t, err)
	assert.Len(t, quarantined, 1, "the unreadable snapshot is kept aside for inspection")
}

func TestCorruptRegistryNamesTheRemedy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	walDir := filepath.Join(t.TempDir(), "wal")

	first := persistenceProvider(t, walDir)
	require.NoError(t, first.Connect(ctx))
	writeCheckpointArtefact(t, first, ctx, "art")
	require.NoError(t, first.Close(ctx))

	snapshotPath := filepath.Join(walDir, "registry", "snapshot.piko")
	require.FileExists(t, snapshotPath)
	require.NoError(t, os.WriteFile(snapshotPath, []byte("written by something else entirely"), 0o600))

	second := persistenceProvider(t, walDir)
	err := second.Connect(ctx)
	require.Error(t, err, "an unreadable registry must not be silently discarded")
	assert.Contains(t, err.Error(), filepath.Join(walDir, "registry"), "the error names the directory")
	assert.Contains(t, err.Error(), "piko build", "the error names the remedy")
}
