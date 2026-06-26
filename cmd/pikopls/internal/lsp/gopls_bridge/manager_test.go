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

package gopls_bridge

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewManager(t *testing.T) {
	t.Parallel()

	t.Run("is disabled when the bridge is not enabled", func(t *testing.T) {
		t.Parallel()

		manager := NewManager(ManagerConfig{})

		assert.False(t, manager.Enabled())
	})

	t.Run("is disabled when gopls cannot be found", func(t *testing.T) {
		t.Parallel()

		manager := NewManager(ManagerConfig{GoplsPath: filepath.Join(t.TempDir(), "missing-gopls"), Allow: true})

		assert.False(t, manager.Enabled())
	})

	t.Run("is enabled when an explicit executable exists", func(t *testing.T) {
		t.Parallel()

		executable, err := os.Executable()
		require.NoError(t, err)

		manager := NewManager(ManagerConfig{GoplsPath: executable, Allow: true})

		assert.True(t, manager.Enabled())
		assert.Equal(t, executable, manager.resolvedGoplsPath())
	})
}

func TestManagerAcquireWhenDisabled(t *testing.T) {
	t.Parallel()

	manager := NewManager(ManagerConfig{})

	child, err := manager.Acquire(context.Background(), t.TempDir())

	assert.Nil(t, child)
	assert.True(t, errors.Is(err, ErrGoplsUnavailable))
}

func TestManagerCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	manager := NewManager(ManagerConfig{})

	require.NoError(t, manager.Close(context.Background()))
	require.NoError(t, manager.Close(context.Background()))
	assert.False(t, manager.Enabled())
}
