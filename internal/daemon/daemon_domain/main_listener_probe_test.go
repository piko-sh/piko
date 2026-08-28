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

package daemon_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func TestMainListenerProbe(t *testing.T) {
	t.Parallel()

	t.Run("readiness fails while the listener has not bound", func(t *testing.T) {
		t.Parallel()

		probe := NewMainListenerProbe(make(chan struct{}), nil)

		status := probe.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State,
			"an orchestrator must not route traffic to a process whose listener never bound")
		assert.Contains(t, status.Message, "has not bound")
	})

	t.Run("liveness passes while the listener has not bound", func(t *testing.T) {
		t.Parallel()

		probe := NewMainListenerProbe(make(chan struct{}), nil)

		status := probe.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State,
			"a process still binding is alive, and killing it would only restart it into the "+
				"same conflict")
	})

	t.Run("readiness passes once the listener has bound", func(t *testing.T) {
		t.Parallel()

		bound := make(chan struct{})
		close(bound)
		probe := NewMainListenerProbe(bound, nil)

		status := probe.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		assert.Equal(t, mainListenerProbeName, status.Name)
	})
}
