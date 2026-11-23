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

package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestGoHostEvidencePlansWithoutCrediting(t *testing.T) {
	deploy := generate.New("embedded-deploy", 7, 0)
	evidence := GoHostEvidence(&deploy, &Observation{})
	assert.Equal(t, HostEvidence{Planned: 1, Executed: 0, Verified: 0}, evidence["binary-only-deployment"])

	topology := generate.New("go-topology", 7, 0)
	evidence = GoHostEvidence(&topology, &Observation{})
	assert.Equal(t, uint64(topologyInstanceCount), evidence["fresh-service-instance"].Planned)
	assert.Zero(t, evidence["fresh-service-instance"].Executed)

	pages := generate.New("pages", 7, 0)
	assert.Nil(t, GoHostEvidence(&pages, &Observation{}))
}

func TestCountTopologyCountsExecutedOperations(t *testing.T) {
	receipt := &GoTopologyReceipt{
		Workspace: "workspace",
		Module:    "module",
		Instances: []GoTopologyInstance{{Reads: []ProbeObservation{{}, {}}, ActionStatus: 200}, {}},
	}

	tally := countTopology(receipt)

	assert.Equal(t, uint64(2), tally.resolution)
	assert.Equal(t, uint64(1), tally.hosts)
	assert.Equal(t, uint64(1), tally.actions)
	assert.Equal(t, uint64(2), tally.pages)
	assert.Equal(t, topologyTally{}, countTopology(nil))
}
