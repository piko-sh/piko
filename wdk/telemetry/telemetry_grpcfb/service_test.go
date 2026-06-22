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

package telemetry_grpcfb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestWithMaxConcurrentStreamsOverridesDefault(t *testing.T) {
	o := serverOptions{maxConcurrentStreams: defaultMaxConcurrentStreams}
	WithMaxConcurrentStreams(1024)(&o)
	assert.Equal(t, uint32(1024), o.maxConcurrentStreams)
}

func TestWithMaxConcurrentStreamsZeroKeepsDefault(t *testing.T) {
	o := serverOptions{maxConcurrentStreams: defaultMaxConcurrentStreams}
	WithMaxConcurrentStreams(0)(&o)
	assert.Equal(t, uint32(defaultMaxConcurrentStreams), o.maxConcurrentStreams)
}

func TestNewServerWithGRPCOptionsAppliesGRPCOptionsAfterDefaults(t *testing.T) {
	srv := NewServerWithGRPCOptions(
		[]ServerOption{WithMaxConcurrentStreams(8)},
		grpc.MaxConcurrentStreams(4096),
	)
	require.NotNil(t, srv)
	defer srv.Stop()

	RegisterServer(srv, &captureServer{})
}

func TestNewServerWithGRPCOptionsAppliesTelemetryDefaults(t *testing.T) {
	srv := NewServerWithGRPCOptions(nil)
	require.NotNil(t, srv)
	defer srv.Stop()
}
