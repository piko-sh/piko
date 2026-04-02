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
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartRollingTraceRecorder_StartsConfiguredRecorder(t *testing.T) {
	t.Parallel()

	recorder := &recordingTraceRecorder{enabled: true, data: []byte("trace")}
	rollingTrace, err := startRollingTraceRecorder(
		Config{EnableRollingTrace: true, RollingTraceMinAge: 3 * time.Second, RollingTraceMaxBytes: 64 * 1024},
		recorder,
	)
	require.NoError(t, err)
	require.NotNil(t, rollingTrace)
	assert.True(t, recorder.started)
	assert.Equal(t, 3*time.Second, rollingTrace.minAge)
	assert.EqualValues(t, 64*1024, rollingTrace.maxBytes)

	var buffer bytes.Buffer
	_, err = rollingTrace.WriteTo(&buffer)
	require.NoError(t, err)
	assert.Equal(t, "trace", buffer.String())
}

func TestStartRollingTraceRecorder_PropagatesStartErrors(t *testing.T) {
	t.Parallel()

	rollingTrace, err := startRollingTraceRecorder(
		Config{EnableRollingTrace: true},
		&recordingTraceRecorder{startErr: errors.New("start failed")},
	)
	require.Error(t, err)
	assert.Nil(t, rollingTrace)
}

func TestRollingTraceRecorder_WriteToDisabledRecorderReturnsError(t *testing.T) {
	t.Parallel()

	rollingTrace := &rollingTraceRecorder{}
	_, err := rollingTrace.WriteTo(&bytes.Buffer{})
	require.ErrorIs(t, err, ErrRollingTraceDisabled)
}

type recordingTraceRecorder struct {
	enabled  bool
	started  bool
	data     []byte
	startErr error
}

func (r *recordingTraceRecorder) Enabled() bool { return r.enabled }

func (r *recordingTraceRecorder) Start() error {
	r.started = true
	return r.startErr
}

func (r *recordingTraceRecorder) Stop() {}

func (r *recordingTraceRecorder) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(r.data)
	return int64(n), err
}
