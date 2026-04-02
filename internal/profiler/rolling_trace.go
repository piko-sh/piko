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
	"errors"
	"fmt"
	"io"
	runtimeTrace "runtime/trace"
	"time"
)

// traceRecorder captures the runtime/trace.FlightRecorder surface so tests can
// exercise rolling trace behaviour without depending on the global runtime hook.
type traceRecorder interface {
	// Enabled reports whether the recorder is actively capturing trace data.
	Enabled() bool

	// Start begins trace capture.
	Start() error

	// Stop ends trace capture.
	Stop()

	// WriteTo writes the captured trace data to the provided writer.
	WriteTo(w io.Writer) (n int64, err error)
}

// rollingTraceRecorder manages an optional runtime/trace.FlightRecorder.
type rollingTraceRecorder struct {
	// recorder holds the underlying trace recorder implementation.
	recorder traceRecorder

	// minAge is the minimum duration of trace data to retain in the rolling buffer.
	minAge time.Duration

	// maxBytes is the maximum number of bytes the rolling trace buffer may use.
	maxBytes uint64
}

// newRollingTraceRecorder creates and starts a rolling trace recorder when the
// config enables it.
//
// Takes config (Config) which provides rolling trace settings.
//
// Returns *rollingTraceRecorder which is the started recorder, or nil when
// rolling trace is disabled.
// Returns error when the recorder cannot be started.
func newRollingTraceRecorder(config Config) (*rollingTraceRecorder, error) {
	if !config.EnableRollingTrace {
		return nil, nil
	}

	recorder := runtimeTrace.NewFlightRecorder(runtimeTrace.FlightRecorderConfig{
		MinAge:   config.RollingTraceMinAge,
		MaxBytes: config.RollingTraceMaxBytes,
	})
	return startRollingTraceRecorder(config, recorder)
}

// startRollingTraceRecorder wraps and starts a trace recorder implementation.
//
// Takes config (Config) which provides min age and max bytes settings.
// Takes recorder (traceRecorder) which is the underlying recorder to start.
//
// Returns *rollingTraceRecorder which is the started recorder.
// Returns error when the recorder fails to start.
func startRollingTraceRecorder(config Config, recorder traceRecorder) (*rollingTraceRecorder, error) {
	if recorder == nil {
		return nil, errors.New("rolling trace recorder must not be nil")
	}

	rollingRecorder := &rollingTraceRecorder{
		recorder: recorder,
		minAge:   config.RollingTraceMinAge,
		maxBytes: config.RollingTraceMaxBytes,
	}

	if err := rollingRecorder.recorder.Start(); err != nil {
		return nil, fmt.Errorf("start rolling trace recorder: %w", err)
	}

	return rollingRecorder, nil
}

// Enabled reports whether rolling trace capture is active.
//
// Returns bool which is true when the recorder is non-nil and actively
// capturing.
func (r *rollingTraceRecorder) Enabled() bool {
	return r != nil && r.recorder != nil && r.recorder.Enabled()
}

// Stop disables rolling trace capture.
func (r *rollingTraceRecorder) Stop() {
	if r == nil || r.recorder == nil {
		return
	}
	r.recorder.Stop()
}

// WriteTo writes the current rolling trace window to the provided writer.
//
// Takes w (io.Writer) which receives the trace data.
//
// Returns int64 which is the number of bytes written.
// Returns error when the recorder is disabled or the write fails.
func (r *rollingTraceRecorder) WriteTo(w io.Writer) (int64, error) {
	if r == nil || r.recorder == nil {
		return 0, ErrRollingTraceDisabled
	}
	return r.recorder.WriteTo(w)
}
