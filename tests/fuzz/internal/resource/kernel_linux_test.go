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

//go:build linux

package resource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeCgroupFile(t *testing.T, directory, name, content string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestCgroupLimitTreatsMaxAndAbsenceAsUnlimited(t *testing.T) {
	directory := t.TempDir()
	writeCgroupFile(t, directory, "memory.max", "4096\n")
	writeCgroupFile(t, directory, "pids.max", "max\n")
	writeCgroupFile(t, directory, "memory.high", "lots\n")

	value, limited, err := cgroupLimit(directory, "memory.max")
	require.NoError(t, err)
	assert.True(t, limited)
	assert.Equal(t, uint64(4096), value)

	_, limited, err = cgroupLimit(directory, "pids.max")
	require.NoError(t, err)
	assert.False(t, limited)

	_, limited, err = cgroupLimit(directory, "absent")
	require.NoError(t, err)
	assert.False(t, limited)

	_, _, err = cgroupLimit(directory, "memory.high")
	require.ErrorContains(t, err, "memory.high")
}

func TestParseQuotaConvertsToCores(t *testing.T) {
	quota, limited, err := parseQuota([]byte("800000 100000\n"))
	require.NoError(t, err)
	assert.True(t, limited)
	assert.InDelta(t, 8.0, quota, 1e-9)

	_, limited, err = parseQuota([]byte("max 100000\n"))
	require.NoError(t, err)
	assert.False(t, limited)

	_, _, err = parseQuota([]byte("100000 0\n"))
	require.ErrorContains(t, err, "invalid period")

	_, _, err = parseQuota([]byte("100000\n"))
	require.Error(t, err)

	_, quoted, err := cgroupQuota(t.TempDir())
	require.NoError(t, err)
	assert.False(t, quoted, "an absent cpu.max sets no quota")
}

func TestParseKeyedPressureAndMemInfo(t *testing.T) {
	events, err := parseKeyed([]byte("low 0\nhigh 3\noom_kill 2\nmalformed\n"))
	require.NoError(t, err)
	assert.Equal(t, map[string]uint64{"low": 0, "high": 3, "oom_kill": 2}, events)

	missing, err := cgroupKeyed(t.TempDir(), "absent")
	require.NoError(t, err)
	assert.Empty(t, missing)

	value, err := parsePressure([]byte("some avg10=12.50 avg60=1.00 avg300=0.00 total=1\nfull avg10=99.00 avg60=0.00 avg300=0.00 total=1\n"))
	require.NoError(t, err)
	assert.InDelta(t, 12.5, value, 1e-9)

	_, err = parsePressure([]byte("full avg10=1.00\n"))
	require.ErrorIs(t, err, errMissingPressure)

	memory, err := parseMemInfo([]byte("MemTotal:       16384 kB\nMemAvailable:    8192 kB\nHugePages_Total:     0\n"))
	require.NoError(t, err)
	assert.Equal(t, uint64(16384<<10), memory["MemTotal"])
	assert.Equal(t, uint64(8192<<10), memory["MemAvailable"])
}

func TestVerifyLimitsRequiresEveryCeiling(t *testing.T) {
	limits := kernelLimits{state: "/state", cpus: 2, memory: 1000}
	enforced := ancestorLimits{
		memory: []uint64{1000}, high: []uint64{800}, swap: []uint64{0}, tasks: []uint64{TasksMax},
		quotas: []float64{2}, headroom: []uint64{1000},
	}
	require.NoError(t, verifyLimits(enforced, []int{0, 1}, limits))

	tooManyCPUs := verifyLimits(enforced, []int{0, 1, 2}, limits)
	require.ErrorContains(t, tooManyCPUs, "CPU cap unverified")

	noQuota := enforced
	noQuota.quotas = nil
	require.ErrorContains(t, verifyLimits(noQuota, []int{0}, limits), "quota=max")

	swapAllowed := enforced
	swapAllowed.swap = []uint64{1}
	require.ErrorIs(t, verifyLimits(swapAllowed, []int{0}, limits), errMemoryCapUnverified)

	looseMemory := enforced
	looseMemory.memory = []uint64{2000}
	require.ErrorIs(t, verifyLimits(looseMemory, []int{0}, limits), errMemoryCapUnverified)
}

func TestDiskUsedCountsFilesAndToleratesAbsence(t *testing.T) {
	root := t.TempDir()
	writeCgroupFile(t, root, "a", "12345")
	require.NoError(t, os.Mkdir(filepath.Join(root, "nested"), 0o700))
	writeCgroupFile(t, filepath.Join(root, "nested"), "b", "678")

	used, err := DiskUsed(t.Context(), root)
	require.NoError(t, err)
	assert.Equal(t, uint64(8), used)

	absent, err := DiskUsed(t.Context(), filepath.Join(root, "absent"))
	require.NoError(t, err)
	assert.Zero(t, absent)

	free, err := DiskFree(filepath.Join(root, "absent", "deeper"))
	require.NoError(t, err)
	assert.Positive(t, free)
}

func TestCheckEnclosingLimitAcceptsNestedGroups(t *testing.T) {
	require.NoError(t, checkEnclosingLimit(""))

	group, err := OwnCgroup()
	if err != nil {
		t.Skip("no cgroup v2 mount: " + err.Error())
	}
	require.NoError(t, checkEnclosingLimit(group))
	require.NoError(t, checkEnclosingLimit(filepath.Dir(group)))
	require.ErrorContains(t, checkEnclosingLimit(group+"-sibling"), "left the pre-existing aggregate limit")
}

func TestOpenRejectsAnUncappedProcessTree(t *testing.T) {
	if _, err := OwnCgroup(); err != nil {
		t.Skip("no cgroup v2 mount: " + err.Error())
	}
	t.Setenv("PIKO_FUZZ_ROOT", t.TempDir())
	t.Setenv("PIKO_FUZZ_STATE", t.TempDir())
	t.Setenv("PIKO_FUZZ_CPUS", "1")
	t.Setenv("PIKO_FUZZ_MEMORY_BYTES", "1")
	t.Setenv("PIKO_FUZZ_MIN_FREE_BYTES", "0")
	t.Setenv("PIKO_FUZZ_DISK_BYTES", "8192")
	t.Setenv("PIKO_FUZZ_ENCLOSING_LIMIT", "")

	_, err := Open(t.Context())

	require.ErrorContains(t, err, "reading kernel resource limits")
}
