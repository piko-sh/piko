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
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/bootstrap/embedregistry"
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

func TestWithEmbeddedDefaults(t *testing.T) {
	embedregistry.Reset()
	t.Cleanup(embedregistry.Reset)

	userOption := func(*bootstrap.Container) {}
	base := []bootstrap.Option{userOption}

	t.Run("prod without a registered payload leaves options untouched", func(t *testing.T) {
		result := withEmbeddedDefaults(RunModeProd, base)
		assert.Len(t, result, 1, "no defaults may be prepended when nothing registered")
	})

	embedregistry.Register(t.Context(), fstest.MapFS{}, []byte("manifest"))

	t.Run("prod with a registered payload prepends both embed options", func(t *testing.T) {
		result := withEmbeddedDefaults(RunModeProd, base)
		assert.Len(t, result, 3, "the embedded folder and manifest options must be prepended together")
	})

	t.Run("dev modes never pick the payload up", func(t *testing.T) {
		assert.Len(t, withEmbeddedDefaults(RunModeDev, base), 1, "dev must keep serving from files")
		assert.Len(t, withEmbeddedDefaults(RunModeDevInterpreted, base), 1, "dev-i must keep serving from files")
	})

	t.Run("user options are applied after the prepended defaults", func(t *testing.T) {
		result := withEmbeddedDefaults(RunModeProd, []bootstrap.Option{bootstrap.WithoutEmbeddedRuntime()})
		container := bootstrap.NewContainer(result...)
		assert.False(t, container.IsEmbeddedMode(), "an explicit WithoutEmbeddedRuntime must override the prepended default")
	})
}

func TestIsRunMode(t *testing.T) {
	t.Parallel()

	t.Run("run modes are recognised", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isRunMode(RunModeProd))
		assert.True(t, isRunMode(RunModeDev))
		assert.True(t, isRunMode(RunModeDevInterpreted))
	})

	t.Run("generate modes are not run modes", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isRunMode(GenerateModeAll))
		assert.False(t, isRunMode(GenerateModeManifest))
		assert.False(t, isRunMode(GenerateModeAssets))
		assert.False(t, isRunMode(GenerateModeSQL))
	})

	t.Run("unknown values are not run modes", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isRunMode(""))
		assert.False(t, isRunMode("production"))
	})
}

func TestIsGenerateMode(t *testing.T) {
	t.Parallel()

	t.Run("generate modes are recognised", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isGenerateMode(GenerateModeAll))
		assert.True(t, isGenerateMode(GenerateModeManifest))
		assert.True(t, isGenerateMode(GenerateModeAssets))
		assert.True(t, isGenerateMode(GenerateModeSQL))
	})

	t.Run("run modes are not generate modes", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isGenerateMode(RunModeProd))
		assert.False(t, isGenerateMode(RunModeDev))
		assert.False(t, isGenerateMode(RunModeDevInterpreted))
	})

	t.Run("unknown values are not generate modes", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isGenerateMode(""))
		assert.False(t, isGenerateMode("build"))
	})
}

func TestSEOProductionSignal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		want *bool
		name string
		mode string
	}{
		{name: "prod is production", mode: RunModeProd, want: new(true)},
		{name: "dev is not production", mode: RunModeDev, want: new(false)},
		{name: "dev-interpreted is not production", mode: RunModeDevInterpreted, want: new(false)},
		{name: "generate all carries no signal", mode: GenerateModeAll, want: nil},
		{name: "generate manifest carries no signal", mode: GenerateModeManifest, want: nil},
		{name: "generate assets carries no signal", mode: GenerateModeAssets, want: nil},
		{name: "generate sql carries no signal", mode: GenerateModeSQL, want: nil},
		{name: "an unknown mode carries no signal", mode: "build", want: nil},
		{name: "an empty mode carries no signal", mode: "", want: nil},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := seoProductionSignal(testCase.mode)

			if testCase.want == nil {
				assert.Nil(t, got, "a mode that says nothing about deployment must leave the signal unset")
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, *testCase.want, *got)
		})
	}
}

func TestGenerate_RejectsNonGenerateModes(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{RunModeProd, RunModeDev, "", "build"} {
		server := &SSRServer{}
		err := server.Generate(context.Background(), mode)

		require.Error(t, err, "mode %q must be rejected", mode)
		assert.ErrorContains(t, err, "is not one of")
	}
}
