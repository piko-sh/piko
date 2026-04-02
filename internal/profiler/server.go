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
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/json"
)

// serverReadHeaderTimeout is the maximum duration the pprof HTTP server
// waits to read request headers.
const serverReadHeaderTimeout = 10 * time.Second

const (
	// serverReadTimeout bounds the total time spent reading a profiler request.
	serverReadTimeout = 10 * time.Second

	// serverWriteTimeout bounds the time spent writing profiler responses.
	serverWriteTimeout = 30 * time.Second

	// serverIdleTimeout bounds how long idle keep-alive connections stay open.
	serverIdleTimeout = 30 * time.Second
)

const (
	// ProfilerStatusPath returns profiler feature and endpoint metadata.
	ProfilerStatusPath = BasePath + "/profiler/status"

	// RollingTracePath downloads the current rolling trace snapshot when
	// rolling trace capture is enabled.
	RollingTracePath = BasePath + "/profiler/trace/recent"
)

// ErrRollingTraceDisabled indicates that rolling trace capture is not enabled.
var ErrRollingTraceDisabled = errors.New("rolling trace capture is not enabled")

// profilerErrorOutput is the fallback sink used when no structured error handler
// has been attached to the server handle. Access via getErrorOutput/setErrorOutput
// for concurrent safety.
var profilerErrorOutput atomic.Value

func init() {
	profilerErrorOutput.Store(io.Writer(os.Stdout))
}

// getErrorOutput returns the current error output writer,
// falling back to stdout if the stored value is not a
// valid writer.
//
// Returns io.Writer which is the active error output sink.
func getErrorOutput() io.Writer {
	if w, ok := profilerErrorOutput.Load().(io.Writer); ok {
		return w
	}
	return os.Stdout
}

// setErrorOutput replaces the error output writer used
// by the profiler server.
//
// Takes w (io.Writer) which is the new error output sink.
func setErrorOutput(w io.Writer) {
	profilerErrorOutput.Store(w)
}

// RollingTraceStatus describes the rolling trace feature exposed by the
// profiler HTTP server.
type RollingTraceStatus struct {
	// MinAge is the minimum age of trace data retained, formatted as a duration string.
	MinAge string `json:"min_age"`

	// DownloadPath is the URL path for downloading the rolling trace snapshot.
	DownloadPath string `json:"download_path,omitempty"`

	// MaxBytes is the maximum size of the rolling trace buffer in bytes.
	MaxBytes uint64 `json:"max_bytes"`

	// Enabled indicates whether rolling trace capture is active.
	Enabled bool `json:"enabled"`
}

// ServerStatus describes the profiler HTTP server capabilities.
type ServerStatus struct {
	// PprofBasePath is the URL path prefix for standard pprof endpoints.
	PprofBasePath string `json:"pprof_base_path"`

	// StatusPath is the URL path for querying profiler status metadata.
	StatusPath string `json:"status_path"`

	// RollingTrace holds the rolling trace feature status and settings.
	RollingTrace RollingTraceStatus `json:"rolling_trace"`
}

// ServerHandle wraps the HTTP server and any optional runtime profiler
// extensions that need coordinated shutdown.
type ServerHandle struct {
	// server is the pprof HTTP server instance.
	server *http.Server

	// rollingTrace is the optional rolling trace recorder, nil when disabled.
	rollingTrace *rollingTraceRecorder

	// onError is the optional callback invoked when an asynchronous server error occurs.
	onError func(error)
}

// StartServer creates and starts a pprof HTTP server on the address specified
// in config. The server runs in a background goroutine and exposes the standard
// pprof endpoints under /_piko/debug/pprof/.
//
// Takes config (Config) which provides Port and BindAddress.
//
// Returns *ServerHandle which can be used for graceful shutdown.
// Returns error when optional rolling trace capture could not be started.
func StartServer(config Config) (*ServerHandle, error) {
	rollingTrace, err := newRollingTraceRecorder(config)
	if err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(config.BindAddress, strconv.Itoa(config.Port))
	server := &http.Server{
		Addr:              addr,
		Handler:           newServerHandler(rollingTrace),
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
		ReadHeaderTimeout: serverReadHeaderTimeout,
	}

	handle := &ServerHandle{
		server:       server,
		rollingTrace: rollingTrace,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			handle.reportError(fmt.Errorf("profiling server failed: %w", err))
		}
	}()

	return handle, nil
}

// Shutdown gracefully stops the profiler HTTP server and any auxiliary rolling
// trace recorder.
//
// Returns error when the HTTP server shutdown fails.
func (s *ServerHandle) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}

	if s.rollingTrace != nil {
		defer s.rollingTrace.Stop()
	}

	if s.server == nil {
		return nil
	}

	return s.server.Shutdown(ctx)
}

// SetErrorHandler configures an optional callback for asynchronous profiler
// server errors.
//
// Takes fn (func(error)) which is the error callback to invoke when the
// server encounters an asynchronous error.
func (s *ServerHandle) SetErrorHandler(fn func(error)) {
	if s == nil {
		return
	}
	s.onError = fn
}

// reportError dispatches an error to the configured error
// handler, or writes it to the fallback error output if
// no handler is set.
//
// Takes err (error) which is the error to report.
func (s *ServerHandle) reportError(err error) {
	if err == nil {
		return
	}
	if s != nil && s.onError != nil {
		s.onError(err)
		return
	}
	_, _ = fmt.Fprintf(getErrorOutput(), "profiling server error: %v\n", err)
}

// newServerHandler builds the HTTP handler mux that
// serves pprof endpoints, profiler status, and rolling
// trace downloads.
//
// Takes rollingTrace (*rollingTraceRecorder) which is
// the optional rolling trace recorder (may be nil).
//
// Returns http.Handler which serves all profiler routes.
func newServerHandler(rollingTrace *rollingTraceRecorder) http.Handler {
	pprofMux := http.NewServeMux()
	pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
	pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	mux := http.NewServeMux()
	mux.HandleFunc(ProfilerStatusPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeProfilerStatus(w, rollingTrace)
	})
	mux.HandleFunc(RollingTracePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeRollingTrace(w, rollingTrace)
	})
	mux.Handle(BasePath+"/", http.StripPrefix(BasePath, pprofMux))

	return mux
}

// writeProfilerStatus serialises the current profiler
// status as JSON and writes it to the HTTP response.
//
// Takes w (http.ResponseWriter) which receives the JSON
// response.
// Takes rollingTrace (*rollingTraceRecorder) which
// provides the rolling trace state.
func writeProfilerStatus(w http.ResponseWriter, rollingTrace *rollingTraceRecorder) {
	data, err := json.Marshal(serverStatus(rollingTrace))
	if err != nil {
		http.Error(w, fmt.Sprintf("encode profiler status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

// serverStatus builds a ServerStatus snapshot describing
// the profiler server's current capabilities and rolling
// trace state.
//
// Takes rollingTrace (*rollingTraceRecorder) which
// provides the rolling trace state.
//
// Returns ServerStatus which describes the server's
// current capabilities.
func serverStatus(rollingTrace *rollingTraceRecorder) ServerStatus {
	status := ServerStatus{
		PprofBasePath: BasePath + "/debug/pprof",
		StatusPath:    ProfilerStatusPath,
		RollingTrace: RollingTraceStatus{
			Enabled: false,
		},
	}

	if rollingTrace != nil && rollingTrace.Enabled() {
		status.RollingTrace.Enabled = true
		status.RollingTrace.MinAge = rollingTrace.minAge.String()
		status.RollingTrace.MaxBytes = rollingTrace.maxBytes
		status.RollingTrace.DownloadPath = RollingTracePath
	}

	return status
}

// writeRollingTrace streams the current rolling trace
// snapshot to the HTTP response as an octet-stream
// attachment.
//
// Takes w (http.ResponseWriter) which receives the trace
// data.
// Takes rollingTrace (*rollingTraceRecorder) which
// supplies the trace snapshot.
func writeRollingTrace(w http.ResponseWriter, rollingTrace *rollingTraceRecorder) {
	if rollingTrace == nil {
		http.Error(w, "rolling trace capture is not enabled", http.StatusNotFound)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="rolling-trace.out"`)

	if _, err := rollingTrace.WriteTo(w); err != nil {
		_, _ = fmt.Fprintf(getErrorOutput(), "rolling trace write error: %v\n", err)
	}
}

// ServerAddress returns the formatted address string for the pprof server.
//
// Takes config (Config) which provides BindAddress and Port.
//
// Returns string which is the address in "host:port" format.
func ServerAddress(config Config) string {
	return net.JoinHostPort(config.BindAddress, strconv.Itoa(config.Port))
}
