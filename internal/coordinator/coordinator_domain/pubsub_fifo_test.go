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

package coordinator_domain

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func receiveNotification(t *testing.T, notifications <-chan BuildNotification) BuildNotification {
	t.Helper()
	select {
	case notification, ok := <-notifications:
		require.True(t, ok, "notification channel closed early")
		return notification
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a build notification")
		return BuildNotification{}
	}
}

func TestPublish_DeliversEveryNotificationInOrder(t *testing.T) {
	t.Parallel()

	service := newTargetedTestService(&mockAnnotator{})
	notifications, unsubscribe := service.Subscribe("fifo")
	t.Cleanup(unsubscribe)

	const published = 20
	for index := range published {
		service.publish(context.Background(), BuildNotification{CausationID: strconv.Itoa(index)})
	}

	for index := range published {
		assert.Equal(t, strconv.Itoa(index), receiveNotification(t, notifications).CausationID,
			"a subscriber that is briefly busy must still see every build, in publish order")
	}
}

func TestPublish_OrderIsKeptAcrossTheDirectAndQueuedPaths(t *testing.T) {
	t.Parallel()

	service := newTargetedTestService(&mockAnnotator{})
	notifications, unsubscribe := service.Subscribe("interleaved")
	t.Cleanup(unsubscribe)

	next := 0
	for round := range 5 {
		for range round + 1 {
			service.publish(context.Background(), BuildNotification{CausationID: strconv.Itoa(next)})
			next++
		}
		for expected := next - round - 1; expected < next; expected++ {
			assert.Equal(t, strconv.Itoa(expected), receiveNotification(t, notifications).CausationID)
		}
	}
}

func TestUnsubscribe_StopsAPumpWithQueuedNotifications(t *testing.T) {
	t.Parallel()

	service := newTargetedTestService(&mockAnnotator{})
	notifications, unsubscribe := service.Subscribe("abandoned")
	for index := range 10 {
		service.publish(context.Background(), BuildNotification{CausationID: fmt.Sprint(index)})
	}

	unsubscribe()

	drained := 0
	for range notifications {
		drained++
	}
	assert.LessOrEqual(t, drained, 1, "only the notification already in the channel survives unsubscribing")

	assert.NotPanics(t, func() {
		service.publish(context.Background(), BuildNotification{CausationID: "after"})
	})
}

func TestShutdown_StopsNotificationPumps(t *testing.T) {
	t.Parallel()

	service := newTargetedTestService(&mockAnnotator{})
	_, unsubscribe := service.Subscribe("never-reads")
	t.Cleanup(unsubscribe)
	for index := range 10 {
		service.publish(context.Background(), BuildNotification{CausationID: fmt.Sprint(index)})
	}

	service.stopNotificationPumps()

	service.subMutex.RLock()
	defer service.subMutex.RUnlock()
	for _, sub := range service.subscribers {
		sub.pump.mu.Lock()
		assert.False(t, sub.pump.active, "no delivery goroutine may outlive the coordinator")
		sub.pump.mu.Unlock()
	}
}
