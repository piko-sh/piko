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

package piko

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/pikotest/pikotest_dto"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/templater/templater_dto"
)

type fakeLifecycleComponent struct {
	recorder *callRecorder
	name     string
}

func (f *fakeLifecycleComponent) OnStart(_ context.Context) error {
	return nil
}

func (f *fakeLifecycleComponent) OnStop(_ context.Context) error {
	f.recorder.append(f.name)
	return nil
}

func (f *fakeLifecycleComponent) Name() string {
	return f.name
}

type callRecorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *callRecorder) append(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, name)
}

func (r *callRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.calls))
	copy(out, r.calls)
	return out
}

func TestEnsurePikoInternalDir(t *testing.T) {
	t.Parallel()

	t.Run("creates internal directory", func(t *testing.T) {
		t.Parallel()
		baseDir := t.TempDir()

		err := ensurePikoInternalDir(baseDir, ".piko")

		require.NoError(t, err)
	})

	t.Run("returns error for invalid base directory", func(t *testing.T) {
		t.Parallel()

		err := ensurePikoInternalDir("/nonexistent/path/that/cannot/exist", ".piko")

		assert.Error(t, err)
	})
}

func TestRegisterLifecycleShutdownHooksOrder(t *testing.T) {
	shutdown.Reset()
	t.Cleanup(shutdown.Reset)

	recorder := &callRecorder{}

	server := &SSRServer{
		lifecycleComponents: []LifecycleComponent{
			&fakeLifecycleComponent{name: "A", recorder: recorder},
			&fakeLifecycleComponent{name: "B", recorder: recorder},
			&fakeLifecycleComponent{name: "C", recorder: recorder},
		},
	}

	ctx, cancel := context.WithCancelCause(t.Context())
	t.Cleanup(func() { cancel(nil) })

	server.registerLifecycleShutdownHooks(ctx)
	shutdown.Cleanup(ctx, 5*time.Second)

	assert.Equal(t, []string{"C", "B", "A"}, recorder.snapshot(),
		"components must be stopped in reverse-of-registration order (LIFO)")
}

func TestRegisterLifecycleShutdownHooksEmptySliceNoop(t *testing.T) {
	shutdown.Reset()
	t.Cleanup(shutdown.Reset)

	server := &SSRServer{}
	recorder := &callRecorder{}

	server.registerLifecycleShutdownHooks(t.Context())

	shutdown.Register(t.Context(), "sentinel", func(_ context.Context) error {
		recorder.append("sentinel")
		return nil
	})

	shutdown.Cleanup(t.Context(), 5*time.Second)

	assert.Equal(t, []string{"sentinel"}, recorder.snapshot(),
		"only the sentinel hook should run; no lifecycle hooks were registered")
}

func TestRunRegistersLifecycleHooksBeforeSpawningSignalListener(t *testing.T) {
	shutdown.Reset()
	t.Cleanup(shutdown.Reset)

	recorder := &callRecorder{}

	server := &SSRServer{
		lifecycleComponents: []LifecycleComponent{
			&fakeLifecycleComponent{name: "DB", recorder: recorder},
			&fakeLifecycleComponent{name: "Cache", recorder: recorder},
		},
	}

	server.registerLifecycleShutdownHooks(t.Context())

	shutdown.Cleanup(t.Context(), 5*time.Second)

	got := recorder.snapshot()
	require.Len(t, got, 2, "both registered hooks must run during Cleanup")
	assert.Equal(t, []string{"Cache", "DB"}, got,
		"a SIGTERM during the early-startup window must invoke OnStop hooks")
}

func TestPikotestWithRendererDoesNotPanic(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	option := WithRenderer(nil)
	require.NotNil(t, option, "WithRenderer must return a non-nil ComponentOption")

	cfg := pikotest_dto.DefaultComponentConfig()

	require.NotPanics(t, func() { option(&cfg) },
		"applying the deprecated WithRenderer option must not panic")

	logged := buf.String()
	assert.Contains(t, logged, "import cycle",
		"slog message must explain the import-cycle reason for the stub")
	assert.Contains(t, logged, "pikotest_domain.WithRenderer",
		"slog message must guide callers to the correct API")
	assert.Nil(t, cfg.Renderer,
		"the stub option must not configure a renderer")
}

func TestGetErrorContext(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil request", func(t *testing.T) {
		t.Parallel()

		result := GetErrorContext(nil)
		assert.Nil(t, result)
	})

	t.Run("returns nil when no error context in request", func(t *testing.T) {
		t.Parallel()

		rd := templater_dto.NewRequestDataBuilder().Build()
		defer rd.Release()

		result := GetErrorContext(rd)
		assert.Nil(t, result)
	})

	t.Run("returns error context when present", func(t *testing.T) {
		t.Parallel()

		epc := daemon_dto.ErrorPageContext{
			StatusCode:   404,
			Message:      "page not found",
			OriginalPath: "/missing",
		}
		ctx := daemon_dto.WithErrorPageContext(t.Context(), epc)
		rd := templater_dto.NewRequestDataBuilder().
			WithContext(ctx).
			Build()
		defer rd.Release()

		result := GetErrorContext(rd)
		require.NotNil(t, result)
		assert.Equal(t, 404, result.StatusCode)
		assert.Equal(t, "page not found", result.Message)
		assert.Equal(t, "/missing", result.OriginalPath)
	})
}
