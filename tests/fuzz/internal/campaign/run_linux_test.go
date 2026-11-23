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
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/process"
)

func TestExecuteInterruptKeepsTheExactActiveCase(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "ready")
	report := cleanReport()
	report.Options.Cases = 2
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := make(chan struct{})
	var hashes []string
	var childErr error
	h := Harness{Budget: testBudget{}, Output: &strings.Builder{}, Progress: &strings.Builder{}, trial: func(ctx context.Context, c *casefile.Case, _, _ string) Result {
		hashes = append(hashes, c.Hash())
		if c.Index == 1 && len(hashes) == 2 {
			close(started)
			_, childErr = process.Run(ctx, dir, nil, "sh", "-c", `sleep 60 & echo "$!" > "$1"; wait`, "phase", marker)
			return Result{Kind: "interrupted", Target: c.Target}
		}
		return Result{Kind: "pass", Target: c.Target}
	}}
	stopped := make(chan error, 1)
	go func() { stopped <- h.Execute(ctx, dir, &report) }()
	<-started
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(marker)
		return err == nil && strings.HasSuffix(string(data), "\n")
	}, time.Minute, 10*time.Millisecond, "child never reported its process identifier")
	cancel()
	require.ErrorIs(t, <-stopped, context.Canceled)
	require.Error(t, childErr, "interrupted process reported success")
	id, err := os.ReadFile(marker)
	require.NoError(t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(id)))
	require.NoError(t, err)
	require.Positive(t, pid)
	stat, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err == nil {
		assert.Contains(t, string(stat), ") Z ", "child %d still running after cancellation", pid)
	}
	saved, err := Load(dir)
	require.NoError(t, err)
	require.Len(t, hashes, 2)
	assert.Equal(t, uint64(1), saved.Next)
	assert.Equal(t, hashes[1], saved.Active)
	assert.Equal(t, uint64(1), saved.Metrics.Cases)
	require.NoError(t, h.Execute(t.Context(), dir, &saved))
	require.Len(t, hashes, 3)
	assert.Equal(t, hashes[1], hashes[2])
	assert.Equal(t, uint64(2), saved.Next)
	assert.Equal(t, uint64(2), saved.Metrics.Cases)
	assert.Empty(t, saved.Active)
	_, err = os.Stat(filepath.Join(dir, "active"))
	assert.ErrorIs(t, err, fs.ErrNotExist, "committed input not cleaned up")
}
