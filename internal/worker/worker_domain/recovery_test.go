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

package worker_domain

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/clock"
)

type fakeRecoveryStore struct {
	lastOlderThan time.Duration
	reset         int
	calls         int
}

func (f *fakeRecoveryStore) ReclaimStale(_ context.Context, olderThan time.Duration) (int, error) {
	f.calls++
	f.lastOlderThan = olderThan
	return f.reset, nil
}

type fakeRecoveryNotifier struct{ notified int }

func (f *fakeRecoveryNotifier) Notify(context.Context, string) error {
	f.notified++
	return nil
}

func TestRecoverOnStartup_ReclaimsEveryRunningRow(t *testing.T) {
	store := &fakeRecoveryStore{reset: 3}
	notifier := &fakeRecoveryNotifier{}
	recoverer := NewRecoverer(store, notifier, clock.RealClock(), RecoveryConfig{})

	require.NoError(t, recoverer.recoverOnStartup(context.Background()), "recoverOnStartup")
	require.Zero(t, store.lastOlderThan, "startup reclaims every running row (olderThan == 0)")
	require.Equal(t, 1, notifier.notified, "startup wakes workers once after reclaiming")
}

func TestReclaimOnce_UsesVisibilityTimeout(t *testing.T) {
	store := &fakeRecoveryStore{reset: 1}
	recoverer := NewRecoverer(store, &fakeRecoveryNotifier{}, clock.RealClock(),
		RecoveryConfig{VisibilityTimeout: 2 * time.Second})

	recoverer.reclaimOnce(context.Background())
	require.Equal(t, 2*time.Second, store.lastOlderThan, "periodic reclaim uses the visibility timeout")
}
