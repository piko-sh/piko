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

package resource

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeasuredPhaseCosts(t *testing.T) {
	group, work := t.TempDir(), t.TempDir()
	write := func(path, contents string) {
		t.Helper()
		require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	}
	stat := filepath.Join(group, "cpu.stat")
	write(stat, "usage_usec 50\nthrottled_usec 7\n")
	write(filepath.Join(group, "memory.current"), "100\n")
	expected := errors.New("controlled failure")
	_, cost, err := MeasureCheck(group, work, func() error {
		write(stat, "usage_usec 1050\nthrottled_usec 27\n")
		write(filepath.Join(work, "kept"), "abc")
		return expected
	})
	require.ErrorIs(t, err, expected)
	assert.True(t, cost.CPUKnown)
	assert.Equal(t, uint64(1000), cost.CPUUsec)
	assert.Equal(t, uint64(20), cost.ThrottledUsec)
	assert.True(t, cost.WorkDirKnown)
	assert.Equal(t, uint64(3), cost.EndWorkDirBytes)

	_, cost, err = MeasureCheck(group, work, func() error {
		write(stat, "usage_usec 1\n")
		return nil
	})
	require.NoError(t, err)
	assert.False(t, cost.CPUKnown, "counter reset was credited")
}

func TestCPUUsageReadsCounter(t *testing.T) {
	group := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(group, "cpu.stat"), []byte("usage_usec 9\n"), 0o600))
	usage, ok := CPUUsage(group)
	assert.True(t, ok)
	assert.Equal(t, uint64(9), usage)
	_, ok = CPUUsage("")
	assert.False(t, ok)
}

func TestDiskMeasurementSkipsUnrelatedSymlinkTree(t *testing.T) {
	directory := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "other"), []byte("external"), 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(directory, "outside")))
	size, known := DirBytes(directory)
	assert.True(t, known)
	assert.Zero(t, size, "unexpected external credit")
}

func TestCPUTimesRejectsMalformedCounter(t *testing.T) {
	group := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(group, "cpu.stat"), []byte("usage_usec many\n"), 0o600))

	_, _, ok := cpuTimes(group)

	assert.False(t, ok)
}

func TestCPUTimesRequiresUsageCounter(t *testing.T) {
	group := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(group, "cpu.stat"), []byte("nr_periods 4\nthrottled_usec 3\n"), 0o600))

	usage, throttled, ok := cpuTimes(group)

	assert.False(t, ok)
	assert.Zero(t, usage)
	assert.Equal(t, uint64(3), throttled)
}

func TestCPUTimesReportsMissingAccounting(t *testing.T) {
	_, _, ok := cpuTimes(t.TempDir())

	assert.False(t, ok)
}

func TestReadCgroupFileReportsMissingGroup(t *testing.T) {
	_, err := readCgroupFile(filepath.Join(t.TempDir(), "absent"), "memory.current")

	assert.ErrorContains(t, err, "open cgroup")
}

func TestDirBytesReportsUnknownForMissingOrEmptyDirectory(t *testing.T) {
	_, known := DirBytes("")
	assert.False(t, known)

	_, known = DirBytes(filepath.Join(t.TempDir(), "absent"))
	assert.False(t, known)
}
