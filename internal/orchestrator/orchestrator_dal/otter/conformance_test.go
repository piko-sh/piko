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

package otter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/dalconformance"
	orchestrator_otter "piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/persistence"
	clockpkg "piko.sh/piko/wdk/clock"
)

var (
	otterGaps = map[string]string{}
)

func TestConformance_OtterMemory(t *testing.T) {
	dalconformance.Run(t, dalconformance.Backend{
		Open: func(t *testing.T, clock clockpkg.Clock) dalconformance.Instance {
			dal, err := orchestrator_otter.NewOtterDAL(orchestrator_otter.Config{}, orchestrator_otter.WithClock(clock))
			require.NoError(t, err)
			t.Cleanup(func() { _ = dal.Close() })
			return dalconformance.Instance{Store: dal}
		},
		Gaps: otterGaps,
	})
}

func TestConformance_OtterWAL(t *testing.T) {
	dalconformance.Run(t, dalconformance.Backend{
		Open: func(t *testing.T, clock clockpkg.Clock) dalconformance.Instance {
			dir := t.TempDir()
			store, closeStore := openWALStore(t, dir, clock)
			t.Cleanup(func() { closeStore() })
			return dalconformance.Instance{
				Store: store,
				Reopen: func(t *testing.T) orchestrator_domain.TaskStore {
					closeStore()
					reopened, closeReopened := openWALStore(t, dir, clock)
					closeStore = closeReopened
					return reopened
				},
			}
		},
		Gaps: otterGaps,
	})
}

func openWALStore(t *testing.T, dir string, clock clockpkg.Clock) (orchestrator_domain.TaskStore, func()) {
	t.Helper()
	provider := persistence.NewProvider(persistence.Config{
		Persistence: &persistence.PersistenceProviderConfig{WALDir: dir, Enabled: true},
		Clock:       clock,
	})
	require.NoError(t, provider.Connect(context.Background()))
	factory, err := provider.OrchestratorDALFactory()
	require.NoError(t, err)
	dal, err := factory.NewOrchestratorDAL()
	require.NoError(t, err)
	store, ok := dal.(orchestrator_domain.TaskStore)
	require.True(t, ok)

	closed := false
	return store, func() {
		if !closed {
			closed = true
			_ = provider.Close(context.Background())
		}
	}
}
