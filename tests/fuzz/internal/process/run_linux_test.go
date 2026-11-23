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

package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunKillsGrandchildrenAfterNormalExit(t *testing.T) {
	directory := t.TempDir()
	pidFile := filepath.Join(directory, "grandchild.pid")
	output, err := Run(t.Context(), directory, nil, "sh", "-c", "sleep 60 >/dev/null 2>&1 & echo $! > "+pidFile)
	require.NoError(t, err, output)

	contents, err := os.ReadFile(pidFile)
	require.NoError(t, err)
	pid, err := strconv.Atoi(string(bytes.TrimSpace(contents)))
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			t.Logf("kill leftover grandchild %d: %v", pid, err)
		}
	})

	require.Eventually(t, func() bool {
		return processGone(pid)
	}, 5*time.Second, 10*time.Millisecond, "grandchild %d outlived its parent", pid)
}

func TestRunCancellationKillsWholeProcessGroup(t *testing.T) {
	stop := errors.New("test cancelled the child")
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	observer := &pidObserver{cancel: cancel, stop: stop}

	_, err := RunObserved(ctx, t.TempDir(), nil, observer, "sh", "-c", `sleep 60 >/dev/null 2>&1 & echo "$$ $!"; exec sleep 60`)

	require.ErrorIs(t, err, stop)
	pids := observer.reported()
	require.Len(t, pids, 2, "child did not report its process identifiers")
	t.Cleanup(func() {
		for _, pid := range pids {
			if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				t.Logf("kill leftover process %d: %v", pid, err)
			}
		}
	})
	for _, pid := range pids {
		require.Eventually(t, func() bool {
			return processGone(pid)
		}, 5*time.Second, 10*time.Millisecond, "process %d outlived cancellation", pid)
	}
}

type pidObserver struct {
	cancel context.CancelCauseFunc
	stop   error
	data   []byte
	pids   []int
	mu     sync.Mutex
}

func (o *pidObserver) Write(data []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.data = append(o.data, data...)
	line, _, found := bytes.Cut(o.data, []byte("\n"))
	if !found || o.pids != nil {
		return len(data), nil
	}
	o.pids = []int{}
	for field := range strings.FieldsSeq(string(line)) {
		pid, err := strconv.Atoi(field)
		if err == nil {
			o.pids = append(o.pids, pid)
		}
	}
	o.cancel(o.stop)
	return len(data), nil
}

func (o *pidObserver) reported() []int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return slices.Clone(o.pids)
}

func processGone(pid int) bool {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	if err != nil {
		return false
	}
	closing := strings.LastIndexByte(string(stat), ')')
	if closing < 0 {
		return false
	}
	state := strings.TrimSpace(string(stat[closing+1:]))
	return strings.HasPrefix(state, "Z") || strings.HasPrefix(state, "X")
}

func TestKillGroupIgnoresUnstartedCommand(t *testing.T) {
	require.NoError(t, killGroup(exec.Command("true")))
}
