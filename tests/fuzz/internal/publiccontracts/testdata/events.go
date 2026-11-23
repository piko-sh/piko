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
package events_provider_gochannel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"
	events "piko.sh/piko/wdk/events/events_provider_gochannel"
)

var errContractTimeout = errors.New("event contract timed out")

func TestProviderLifecycleContract(t *testing.T) {
	bad := events.DefaultConfig()
	bad.PublishQueueCapacity = -1
	rejected, err := events.NewGoChannelProvider(bad)
	require.True(t, rejected == nil && err != nil, "PIKO_CHECK reject-bad-config: expected constructor error")
	for epoch := range 2 {
		p, err := events.NewGoChannelProvider(events.DefaultConfig())
		require.NoError(t, err)
		defer p.Close()
		ctx, cancel := context.WithTimeoutCause(t.Context(), 4*time.Second, errContractTimeout)
		defer cancel()
		require.False(t, p.Running(), "PIKO_CHECK before-start: provider already running")
		err = p.Start(ctx)
		require.True(t, err == nil && p.Running(), "PIKO_CHECK start: %v", err)
		subCtx, stop := context.WithCancel(ctx)
		ch, err := p.Subscriber().Subscribe(subCtx, "contract")
		require.NoError(t, err)
		want := []byte{byte(epoch), 0, 42}
		require.NoError(t, p.Publisher().Publish("contract", message.NewMessage("local", want)))
		select {
		case m, ok := <-ch:
			require.True(t, ok && string(m.Payload) == string(want), "PIKO_CHECK payload: wrong bytes")
			m.Ack()
		case <-ctx.Done():
			require.FailNow(t, "PIKO_CHECK delivery: timeout", "%v", context.Cause(ctx))
		}
		stop()
		select {
		case _, ok := <-ch:
			require.False(t, ok, "PIKO_CHECK cancel: received extra delivery")
		case <-ctx.Done():
			require.FailNow(t, "PIKO_CHECK cancel: subscription did not close", "%v", context.Cause(ctx))
		}
		err = p.Close()
		require.True(t, err == nil && !p.Running(), "PIKO_CHECK shutdown: %v", err)
		require.ErrorIs(t, p.Start(ctx), events.ErrPublisherClosed, "PIKO_CHECK restart-closed: missing documented closed error")
	}
}
