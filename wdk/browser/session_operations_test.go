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

package browser

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpenSessionCancelledStartup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX stand-in for the Chrome executable")
	}
	ctx, cancel := context.WithTimeoutCause(t.Context(), 5*time.Second, errors.New("cancelled start-up test budget exceeded"))
	defer cancel()
	root := t.TempDir()
	profilesRoot := t.TempDir()
	t.Setenv("TMPDIR", profilesRoot)
	path := filepath.Join(root, "chrome")
	const fakeChrome = `#!/bin/sh
for arg in "$@"; do
  case "$arg" in --user-data-dir=*) printf '%s' "${arg#*=}" > "$TMPDIR/profile";; esac
done
printf '%s' "$TMPDIR" > "$TMPDIR/started"
exec sleep 30
`
	require.NoError(t, os.WriteFile(path, []byte(fakeChrome), 0700))
	inherited := os.Getenv("TMPDIR")
	finished := make(chan error, 1)
	go func() {
		_, err := OpenSession(ctx, BrowserOptions{ChromePath: path, SocketDir: root, Headless: true})
		finished <- err
	}()
	marker := filepath.Join(root, "started")
	require.Eventually(t, func() bool { b, _ := os.ReadFile(marker); return string(b) == root }, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, inherited, os.Getenv("TMPDIR"))
	profile, err := os.ReadFile(filepath.Join(root, "profile"))
	require.NoError(t, err)
	require.Equal(t, profilesRoot, filepath.Dir(string(profile)))
	cancel()
	select {
	case err := <-finished:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		require.FailNow(t, "cancel did not release the launching process")
	}
	require.NoDirExists(t, string(profile))
	_, err = OpenSession(ctx, BrowserOptions{ChromePath: path, SocketDir: root})
	require.ErrorIs(t, err, context.Canceled)
}

func TestNewTabRequiresAPage(t *testing.T) {
	_, err := (&Session{}).NewTab(t.Context())

	require.ErrorIs(t, err, errSessionHasNoPage)
}
