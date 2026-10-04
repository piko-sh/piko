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

// This project stands against fascism, authoritarianism, and all
// forms of oppression. We built this to empower people, not to
// enable those who would strip others of their rights and dignity.

package profiler

import (
	"context"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
)

func freeLoopbackPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", controllerBindAddress+":0")
	require.NoError(t, err)
	address, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	require.NoError(t, listener.Close())

	return address.Port
}

func TestController_RestoresTheMutexFractionItFound(t *testing.T) {
	const applicationFraction = 7
	previous := runtime.SetMutexProfileFraction(applicationFraction)
	t.Cleanup(func() { runtime.SetMutexProfileFraction(previous) })

	controller := NewController()
	status, err := controller.Enable(context.Background(), monitoring_domain.ProfilingEnableOpts{
		Duration:             time.Minute,
		Port:                 freeLoopbackPort(t),
		MutexProfileFraction: 3,
	})
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, 3, runtime.SetMutexProfileFraction(-1), "profiling applies its own fraction")

	disabled, err := controller.Disable(context.Background())
	require.NoError(t, err)
	assert.True(t, disabled)
	assert.Equal(t, applicationFraction, runtime.SetMutexProfileFraction(-1),
		"disabling restores the fraction the application had set")
}
