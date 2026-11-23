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

package runtimecoverage

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/resource"
)

func TestRetentionIncludesOldArtefactsAndHonoursUserDiskBudget(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "older"), make([]byte, 128), 0o600))
	store, err := Open(directory, 512)
	require.NoError(t, err)
	assert.Equal(t, uint64(128), store.limit)
	require.ErrorIs(t, store.Prepare(), resource.ErrLimit)

	store, err = Open(directory, 10<<30)
	require.NoError(t, err)
	assert.Equal(t, uint64(CampaignLimit), store.limit)
	require.NoError(t, store.Prepare())
	assert.Equal(t, uint64(128), store.RetainedBytes())
}

func TestOpenRemovesLeftoverPendingDirectories(t *testing.T) {
	directory := t.TempDir()
	leftover := filepath.Join(directory, pendingPrefix+"abc")
	require.NoError(t, os.Mkdir(leftover, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(leftover, goArtefact), make([]byte, 64), 0o600))
	store, err := Open(directory, 10<<30)
	require.NoError(t, err)
	assert.Zero(t, store.RetainedBytes())
	assert.NoDirExists(t, leftover)
}

func TestPublishRejectsShortHashBeforeWork(t *testing.T) {
	store, err := Open(t.TempDir(), 10<<30)
	require.NoError(t, err)
	_, err = store.Publish(t.Context(), t.TempDir(), Receipt{Target: "render", Mode: "dev", Hash: "abc"})
	require.ErrorIs(t, err, errInvalidReceipt)
	_, err = store.Publish(t.Context(), t.TempDir(), Receipt{Target: "../x", Mode: "dev", Hash: strings.Repeat("a", 64)})
	require.ErrorIs(t, err, errInvalidReceipt)
}

func TestPublishRejectsOversizedRawData(t *testing.T) {
	store, err := Open(t.TempDir(), 10<<30)
	require.NoError(t, err)
	work := t.TempDir()
	raw := filepath.Join(work, "raw")
	require.NoError(t, os.Mkdir(raw, 0o700))
	for index := range maxRawFiles + 1 {
		require.NoError(t, os.WriteFile(filepath.Join(raw, strings.Repeat("f", index+1)), nil, 0o600))
	}
	_, err = store.Publish(t.Context(), work, Receipt{Target: "render", Mode: "dev", Hash: strings.Repeat("a", 64)})
	require.ErrorIs(t, err, resource.ErrLimit)
}

func TestPublishProfileStoresArtefactAtomically(t *testing.T) {
	root := t.TempDir()
	store, err := Open(root, 10<<30)
	require.NoError(t, err)
	work := t.TempDir()
	profile := filepath.Join(work, "go.cover")
	require.NoError(t, os.WriteFile(profile, []byte(testProfile), 0o600))
	browserData, err := json.Marshal(validBrowser())
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(work, JSArtefact), browserData, 0o600))

	receipt, err := store.publishProfile(sandbox(t, work), Receipt{
		Target:          "render",
		Mode:            "dev",
		Hash:            strings.Repeat("ab", 32),
		Index:           7,
		BrowserSelected: true,
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(receipt.Artefact, "render-000007-dev-abababababab-"), receipt.Artefact)
	assert.Equal(t, Counts{Covered: 7, Total: 12}, receipt.GoStatements)
	require.NotNil(t, receipt.JavaScriptFunctions)
	assert.Equal(t, Counts{Covered: 1, Total: 2}, *receipt.JavaScriptFunctions)
	assert.Equal(t, goBasis, receipt.GoBasis)

	artefact := filepath.Join(root, receipt.Artefact)
	for _, name := range []string{goArtefact, JSArtefact, "summary.json"} {
		assert.FileExists(t, filepath.Join(artefact, name))
	}
	summary, err := os.ReadFile(filepath.Join(artefact, "summary.json"))
	require.NoError(t, err)
	var stored Receipt
	require.NoError(t, json.Unmarshal(summary, &stored))
	assert.Equal(t, receipt.Artefact, stored.Artefact)
	assert.Positive(t, store.RetainedBytes())

	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "pending directory left behind")
}

func TestPublishRequiresRawCoverageDirectory(t *testing.T) {
	store, err := Open(t.TempDir(), 10<<30)
	require.NoError(t, err)

	_, err = store.Publish(t.Context(), t.TempDir(), Receipt{Target: "render", Mode: "dev", Hash: strings.Repeat("a", 64)})

	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestPublishProfileRejectsInvalidReceipt(t *testing.T) {
	store, err := Open(t.TempDir(), 10<<30)
	require.NoError(t, err)

	_, err = store.publishProfile(sandbox(t, t.TempDir()), Receipt{Target: "render", Mode: "dev", Hash: "short"})

	require.ErrorIs(t, err, errInvalidReceipt)
}

func TestPublishProfileEnforcesRetention(t *testing.T) {
	root := t.TempDir()
	store, err := Open(root, 64)
	require.NoError(t, err)
	work := t.TempDir()
	profile := filepath.Join(work, "go.cover")
	require.NoError(t, os.WriteFile(profile, []byte(testProfile), 0o600))
	_, err = store.publishProfile(sandbox(t, work), Receipt{Target: "render", Mode: "dev", Hash: strings.Repeat("c", 64)})
	require.ErrorIs(t, err, resource.ErrLimit)
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	assert.Empty(t, entries, "rejected artefact left behind")
}
