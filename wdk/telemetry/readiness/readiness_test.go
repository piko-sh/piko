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

package readiness_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/wdk/telemetry/readiness"
)

func TestStateConstantsCarryStableWireStrings(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "HEALTHY", string(readiness.StateHealthy))
	assert.Equal(t, "DEGRADED", string(readiness.StateDegraded))
	assert.Equal(t, "UNHEALTHY", string(readiness.StateUnhealthy))
}

func TestStateIsValidAcceptsOnlyTheThreeKnownStates(t *testing.T) {
	t.Parallel()

	assert.True(t, readiness.StateHealthy.IsValid())
	assert.True(t, readiness.StateDegraded.IsValid())
	assert.True(t, readiness.StateUnhealthy.IsValid())

	assert.False(t, readiness.State("").IsValid())
	assert.False(t, readiness.State("healthy").IsValid())
	assert.False(t, readiness.State("BOGUS").IsValid())
}

func TestSnapshotTreeRemainsPlainStringDTO(t *testing.T) {
	t.Parallel()

	snapshot := readiness.Snapshot{
		Name:     "readiness",
		State:    readiness.StateDegraded,
		Message:  "one dependency degraded",
		Duration: "1.2ms",
		Dependencies: []readiness.Dependency{
			{
				Name:     "RegistryDatabase",
				State:    readiness.StateHealthy,
				Message:  "",
				Duration: "500us",
				Info: []readiness.InfoEntry{
					{Section: "Overview", Key: "Host", Value: "localhost"},
				},
			},
		},
	}

	assert.Equal(t, "readiness", snapshot.Name)
	assert.Equal(t, readiness.StateDegraded, snapshot.State)
	assert.Equal(t, "one dependency degraded", snapshot.Message)
	assert.Equal(t, "1.2ms", snapshot.Duration)

	deps := snapshot.Dependencies
	assert.Len(t, deps, 1)
	assert.Equal(t, "RegistryDatabase", deps[0].Name)
	assert.Equal(t, readiness.StateHealthy, deps[0].State)
	assert.Equal(t, "500us", deps[0].Duration)

	info := deps[0].Info
	assert.Len(t, info, 1)
	assert.Equal(t, "Overview", info[0].Section)
	assert.Equal(t, "Host", info[0].Key)
	assert.Equal(t, "localhost", info[0].Value)
}
