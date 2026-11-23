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

package campaign

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/resource"
)

func TestSoakGrowthStopsBeforeAdmissionAndCanResume(t *testing.T) {
	directory := t.TempDir()
	files := filepath.Join(directory, "failures", "existing")
	require.NoError(t, os.MkdirAll(files, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(files, "actual"), []byte("01234567890"), 0o600))
	t.Setenv("PIKO_FUZZ_DISK_BYTES", "80")
	report := cleanReport()
	report.Options.Suite = "soak"
	report.Options.Cases = 65
	coordinates := map[[2]uint64]int{}
	harness := Harness{Budget: testBudget{}, Output: io.Discard, Progress: io.Discard, trial: func(_ context.Context, c *casefile.Case, _, _ string) Result {
		coordinates[[2]uint64{c.Seed, c.Index}]++
		return Result{Target: c.Target, Index: c.Index, Kind: "pass"}
	}}
	require.ErrorIs(t, harness.Execute(t.Context(), directory, &report), resource.ErrLimit)
	assert.Equal(t, "resource", report.Status)
	assert.Empty(t, coordinates)
	assert.Zero(t, report.Next)
	assert.Equal(t, uint64(11), report.Retained.FailureBytes)
	assert.Equal(t, uint64(10), report.Retained.SoakLimitBytes)
	t.Setenv("PIKO_FUZZ_DISK_BYTES", "1048576")
	loaded, err := Load(directory)
	require.NoError(t, err)
	require.NoError(t, harness.Execute(t.Context(), directory, &loaded))
	for index := range uint64(65) {
		position := [2]uint64{report.Options.Seed + index/64, index}
		require.Equal(t, 1, coordinates[position], "position %+v", position)
	}
	info, err := os.Stat(filepath.Join(directory, "report.json"))
	require.NoError(t, err)
	assert.Equal(t, uint64(info.Size()), loaded.Retained.CheckpointBytes)
	assert.Equal(t, uint64(11), loaded.Retained.HighWaterBytes)
	assert.Equal(t, uint64(11), loaded.Retained.FailureBytes)
	assert.Equal(t, uint64(65), loaded.Metrics.Cases)
}

func TestRepeatedBundleSizingReplacesOldBytes(t *testing.T) {
	directory := t.TempDir()
	name := filepath.Join(directory, "result.json")
	require.NoError(t, os.WriteFile(name, []byte("large"), 0o600))
	evidence := RetainedEvidence{Known: true}
	evidence.updateBundle(directory, 0, true)
	require.NoError(t, os.WriteFile(name, []byte("ok"), 0o600))
	evidence.updateBundle(directory, 5, true)
	assert.Equal(t, uint64(2), evidence.FailureBytes)
	assert.Equal(t, uint64(5), evidence.HighWaterBytes)
	assert.True(t, evidence.Known)
}

func TestStartRequiresAValidSoakDiskBudget(t *testing.T) {
	for _, value := range []string{"", "not-a-number", "-1", "7"} {
		t.Setenv(diskBudgetVariable, value)
		var evidence RetainedEvidence
		require.ErrorIs(t, evidence.start(t.TempDir(), "soak"), errDiskBudget, "value %q", value)
	}
	t.Setenv(diskBudgetVariable, "")
	var evidence RetainedEvidence
	require.NoError(t, evidence.start(t.TempDir(), "smoke"))
	assert.Zero(t, evidence.SoakLimitBytes)
	t.Setenv(diskBudgetVariable, "80")
	require.NoError(t, evidence.start(t.TempDir(), "soak"))
	assert.Equal(t, uint64(10), evidence.SoakLimitBytes)
}
