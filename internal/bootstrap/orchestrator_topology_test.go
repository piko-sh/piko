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

package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

type topologyStore struct {
	orchestrator_domain.TaskStore
	local bool
}

func (s topologyStore) LocalStore() bool { return s.local }

type topologyBus struct {
	orchestrator_domain.EventBus
	local bool
}

func (b topologyBus) NodeLocal() bool { return b.local }

func TestResolveOrchestratorTopology(t *testing.T) {
	testCases := []struct {
		name       string
		localStore bool
		localBus   bool
		want       orchestrator_domain.TransportMode
		wantErr    bool
	}{
		{name: "in-process store and bus run standalone", localStore: true, localBus: true, want: orchestrator_domain.TransportLocal},
		{name: "shared store and distributed bus run clustered", localStore: false, localBus: false, want: orchestrator_domain.TransportBus},
		{name: "shared store with an in-process bus runs standalone", localStore: false, localBus: true, want: orchestrator_domain.TransportLocal},
		{name: "distributed bus over an in-process store is refused", localStore: true, localBus: false, wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mode, err := resolveOrchestratorTopology(t.Context(),
				topologyStore{local: tc.localStore}, topologyBus{local: tc.localBus})
			if tc.wantErr {
				require.ErrorIs(t, err, errOrchestratorTopology)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, mode)
		})
	}
}
