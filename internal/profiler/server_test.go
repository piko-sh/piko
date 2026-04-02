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

package profiler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerHandler_StatusIncludesRollingTraceDetails(t *testing.T) {
	t.Parallel()

	handler := newServerHandler(&rollingTraceRecorder{
		recorder: fakeTraceRecorder{enabled: true, data: []byte("trace")},
		minAge:   5 * time.Second,
		maxBytes: 128 * 1024,
	},
	)
	server := httptest.NewServer(handler)
	defer server.Close()

	response, err := server.Client().Get(server.URL + ProfilerStatusPath)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	require.Equal(t, http.StatusOK, response.StatusCode)

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	var status ServerStatus
	require.NoError(t, json.Unmarshal(body, &status))
	assert.Equal(t, BasePath+"/debug/pprof", status.PprofBasePath)
	assert.True(t, status.RollingTrace.Enabled)
	assert.Equal(t, "5s", status.RollingTrace.MinAge)
	assert.EqualValues(t, 128*1024, status.RollingTrace.MaxBytes)
	assert.Equal(t, RollingTracePath, status.RollingTrace.DownloadPath)
}

func TestServerHandler_RollingTraceDownloadsSnapshot(t *testing.T) {
	t.Parallel()

	handler := newServerHandler(&rollingTraceRecorder{
		recorder: fakeTraceRecorder{enabled: true, data: []byte("trace-data")},
	},
	)
	server := httptest.NewServer(handler)
	defer server.Close()

	response, err := server.Client().Get(server.URL + RollingTracePath)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	require.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, `attachment; filename="rolling-trace.out"`, response.Header.Get("Content-Disposition"))

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, "trace-data", string(body))
}

func TestServerHandler_RollingTraceReturnsNotFoundWhenDisabled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(newServerHandler(nil))
	defer server.Close()

	response, err := server.Client().Get(server.URL + RollingTracePath)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestServerHandler_RollingTraceWriteErrorReturnsOKWithEmptyBody(t *testing.T) {
	t.Parallel()

	handler := newServerHandler(&rollingTraceRecorder{
		recorder: fakeTraceRecorder{enabled: true, err: errors.New("boom")},
	},
	)
	server := httptest.NewServer(handler)
	defer server.Close()

	response, err := server.Client().Get(server.URL + RollingTracePath)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	assert.Equal(t, http.StatusOK, response.StatusCode)

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Empty(t, body)
}

func TestStartServer_AppliesExpectedTimeouts(t *testing.T) {
	t.Parallel()

	handle, err := StartServer(Config{
		BindAddress: DefaultBindAddress,
		Port:        0,
	})
	require.NoError(t, err)
	require.NotNil(t, handle)

	assert.Equal(t, serverReadTimeout, handle.server.ReadTimeout)
	assert.Equal(t, serverWriteTimeout, handle.server.WriteTimeout)
	assert.Equal(t, serverIdleTimeout, handle.server.IdleTimeout)
	assert.Equal(t, serverReadHeaderTimeout, handle.server.ReadHeaderTimeout)

	require.NoError(t, handle.Shutdown(context.Background()))
}

func TestServerHandle_ReportErrorUsesHandlerWhenConfigured(t *testing.T) {
	t.Parallel()

	handle := &ServerHandle{}

	var got error
	handle.SetErrorHandler(func(err error) {
		got = err
	})

	expected := errors.New("boom")
	handle.reportError(expected)

	require.ErrorIs(t, got, expected)
}

func TestServerHandle_ReportErrorFallsBackToPrintf(t *testing.T) {
	t.Parallel()

	handle := &ServerHandle{}

	output := captureStdout(t, func() {
		handle.reportError(errors.New("boom"))
	})

	assert.Contains(t, output, "profiling server error: boom")
}

type fakeTraceRecorder struct {
	enabled bool
	data    []byte
	err     error
}

func (f fakeTraceRecorder) Enabled() bool { return f.enabled }
func (f fakeTraceRecorder) Start() error  { return nil }
func (f fakeTraceRecorder) Stop()         {}

func (f fakeTraceRecorder) WriteTo(w io.Writer) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	n, err := w.Write(f.data)
	return int64(n), err
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := getErrorOutput()
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	setErrorOutput(writer)

	defer func() {
		setErrorOutput(originalStdout)
	}()

	fn()

	require.NoError(t, writer.Close())

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())

	return strings.TrimSpace(string(data))
}
