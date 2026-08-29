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
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/json"
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

const (
	shutdownSettleWindow = 200 * time.Millisecond
)

func TestStartServer_AppliesExpectedTimeouts(t *testing.T) {
	t.Parallel()

	handle, err := StartServer(t.Context(), Config{
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

func TestProfilerServer_PanicInHandlerIsContained(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/panic", func(_ http.ResponseWriter, _ *http.Request) {
		panic("induced handler panic")
	})
	mux.HandleFunc("/healthy", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	panicResponse, err := server.Client().Get(server.URL + "/panic")
	if err == nil {
		_ = panicResponse.Body.Close()
		assert.NotEqual(t, http.StatusOK, panicResponse.StatusCode,
			"panicking handler should not produce a 200 OK")
	}

	healthyResponse, err := server.Client().Get(server.URL + "/healthy")
	require.NoError(t, err, "subsequent requests must continue to be served after a handler panic")
	defer func() { _ = healthyResponse.Body.Close() }()
	require.Equal(t, http.StatusOK, healthyResponse.StatusCode)
	body, err := io.ReadAll(healthyResponse.Body)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(body))
}

func TestStartServer_ShutsDownWithoutLeavingTheServeGoroutineBehind(t *testing.T) {
	t.Parallel()

	handle, err := StartServer(t.Context(), Config{
		BindAddress: DefaultBindAddress,
		Port:        0,
	})
	require.NoError(t, err)
	require.NotNil(t, handle)

	reported := make(chan error, 1)
	handle.SetErrorHandler(func(err error) { reported <- err })

	require.NoError(t, handle.Shutdown(t.Context()))

	require.Never(t, func() bool { return len(reported) > 0 }, shutdownSettleWindow, time.Millisecond,
		"http.ErrServerClosed is how a listener reports a deliberate stop, so the serve "+
			"goroutine must swallow it rather than raise it as a server failure")
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

func TestStartServer_ReportsABindFailureSynchronously(t *testing.T) {
	t.Parallel()

	held, err := net.Listen("tcp", net.JoinHostPort(DefaultBindAddress, "0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = held.Close() })

	port := listenerPort(t, held)

	handle, err := StartServer(t.Context(), Config{
		BindAddress: DefaultBindAddress,
		Port:        port,
	})

	require.Error(t, err, "a port collision must be reported to the caller, not logged later from a goroutine")
	assert.Nil(t, handle)
}

func TestStartServer_AdvancesToTheNextPortWhenAutoNextPortIsSet(t *testing.T) {
	t.Parallel()

	held, err := net.Listen("tcp", net.JoinHostPort(DefaultBindAddress, "0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = held.Close() })

	port := listenerPort(t, held)

	handle, err := StartServer(t.Context(), Config{
		BindAddress:  DefaultBindAddress,
		Port:         port,
		AutoNextPort: true,
	})
	require.NoError(t, err)
	require.NotNil(t, handle)
	t.Cleanup(func() { _ = handle.Shutdown(t.Context()) })

	_, boundPort, splitErr := net.SplitHostPort(handle.Address())
	require.NoError(t, splitErr)
	assert.NotEqual(t, strconv.Itoa(port), boundPort,
		"the held port must be skipped, and Address must name the port actually bound")
}

func TestStartServer_ReportsTheOSAssignedPortWhenPortIsZero(t *testing.T) {
	t.Parallel()

	handle, err := StartServer(t.Context(), Config{
		BindAddress: DefaultBindAddress,
		Port:        0,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = handle.Shutdown(t.Context()) })

	_, boundPort, splitErr := net.SplitHostPort(handle.Address())
	require.NoError(t, splitErr)
	assert.NotEmpty(t, boundPort)
	assert.NotEqual(t, "0", boundPort, "Address must report the port the OS chose, not the placeholder")
}

func listenerPort(t *testing.T, listener net.Listener) int {
	t.Helper()

	addr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok, "the test listener must be a TCP listener")

	return addr.Port
}

func TestListenForServer_RefusesAPortOutsideTheValidRange(t *testing.T) {
	t.Parallel()

	for _, port := range []int{-1, maxTCPPort + 1} {
		listener, err := listenForServer(t.Context(), Config{
			BindAddress:  DefaultBindAddress,
			Port:         port,
			AutoNextPort: true,
		})

		require.ErrorIs(t, err, errProfilingPortOutOfRange, "port %d", port)
		assert.Nil(t, listener)
	}
}

func TestListenForServer_ReportsABindFailureThatIsNotAPortCollision(t *testing.T) {
	t.Parallel()

	listener, err := listenForServer(t.Context(), Config{
		BindAddress:  "this-host-does-not-resolve.invalid",
		Port:         6060,
		AutoNextPort: true,
	})

	require.Error(t, err)
	assert.NotErrorIs(t, err, errNoProfilingPort,
		"walking a hundred ports on a bad bind address would bury the real error")
	assert.Nil(t, listener)
}

func TestListenForServer_GivesUpAtTheEndOfThePortRange(t *testing.T) {
	t.Parallel()

	held, err := net.Listen("tcp", net.JoinHostPort(DefaultBindAddress, strconv.Itoa(maxTCPPort)))
	if err != nil {
		t.Skip("the last TCP port is not available on this machine")
	}
	t.Cleanup(func() { _ = held.Close() })

	listener, err := listenForServer(t.Context(), Config{
		BindAddress:  DefaultBindAddress,
		Port:         maxTCPPort,
		AutoNextPort: true,
	})

	require.ErrorIs(t, err, errNoProfilingPort)
	assert.Nil(t, listener)
}

func TestListenForServer_StopsWhenTheCallerGoesAway(t *testing.T) {
	t.Parallel()

	held, err := net.Listen("tcp", net.JoinHostPort(DefaultBindAddress, "0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = held.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	listener, err := listenForServer(ctx, Config{
		BindAddress:  DefaultBindAddress,
		Port:         listenerPort(t, held),
		AutoNextPort: true,
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, listener)
}
