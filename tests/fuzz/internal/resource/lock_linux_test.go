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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockAdmitsOneHolderAtATime(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	release, err := Lock()
	require.NoError(t, err)

	_, err = Lock()
	require.ErrorContains(t, err, "another Piko fuzz run holds")

	release()
	again, err := Lock()
	require.NoError(t, err)
	again()
}

func TestLockRejectsSharedDirectory(t *testing.T) {
	shared := t.TempDir()
	require.NoError(t, os.Chmod(shared, 0o777))
	t.Setenv("XDG_RUNTIME_DIR", shared)

	_, err := Lock()

	assert.ErrorIs(t, err, errUnsafeLockDirectory)
}
