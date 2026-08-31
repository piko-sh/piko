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

package logger_state

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	envAttrChildMarker = "PIKO_ENV_ATTR_CHILD"
	envAttrChildPass = "ENV_ATTRS_PRESERVED_OK"
)

type captureHandler struct {
	mu      sync.Mutex
	attrs   []slog.Attr
	records []map[string]string
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	flat := map[string]string{}
	for _, a := range h.attrs {
		flat[a.Key] = a.Value.String()
	}
	r.Attrs(func(a slog.Attr) bool {
		flat[a.Key] = a.Value.String()
		return true
	})
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, flat)
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {

	own := append(append([]slog.Attr{}, h.attrs...), attrs...)

	return &sharedCapture{parent: h, own: own}
}

func (h *captureHandler) WithGroup(string) slog.Handler { return h }

type sharedCapture struct {
	parent *captureHandler
	own    []slog.Attr
}

func (h *sharedCapture) Enabled(context.Context, slog.Level) bool { return true }

func (h *sharedCapture) Handle(_ context.Context, r slog.Record) error {
	flat := map[string]string{}
	for _, a := range h.own {
		flat[a.Key] = a.Value.String()
	}
	r.Attrs(func(a slog.Attr) bool {
		flat[a.Key] = a.Value.String()
		return true
	})
	h.parent.mu.Lock()
	defer h.parent.mu.Unlock()
	h.parent.records = append(h.parent.records, flat)
	return nil
}

func (h *sharedCapture) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &sharedCapture{parent: h.parent, own: slices.Concat(h.own, attrs)}
}

func (h *sharedCapture) WithGroup(string) slog.Handler { return h }

func TestAddHandler_PreservesEnvironmentAttrs(t *testing.T) {
	if os.Getenv(envAttrChildMarker) == "1" {
		runEnvironmentAttrChild(t)
		return
	}

	cmd := exec.CommandContext(t.Context(), os.Args[0],
		"-test.run=TestAddHandler_PreservesEnvironmentAttrs", "-test.v")
	cmd.Env = append(os.Environ(),
		envAttrChildMarker+"=1",
		"PIKO_SERVICE_VERSION=9.9.9-test",
		"PIKO_ENVIRONMENT=staging",
	)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "child process output:\n%s", out)
	require.Containsf(t, string(out), envAttrChildPass, "child process output:\n%s", out)
}

func runEnvironmentAttrChild(t *testing.T) {
	t.Helper()

	sink := &captureHandler{}
	AddHandler(sink, nil)
	t.Cleanup(func() { ResetState() })

	slog.Default().Info("hello")

	sink.mu.Lock()
	defer sink.mu.Unlock()
	require.NotEmpty(t, sink.records, "the destination handler receives the record")

	got := sink.records[len(sink.records)-1]
	for key, want := range map[string]string{
		"service.version":             "9.9.9-test",
		"deployment.environment.name": "staging",
	} {
		require.Equalf(t, want, got[key], "record carries %q; keys present: %v", key, keysOf(got))
	}
	t.Log(envAttrChildPass)
}

func keysOf(record map[string]string) []string {
	return slices.Sorted(maps.Keys(record))
}
