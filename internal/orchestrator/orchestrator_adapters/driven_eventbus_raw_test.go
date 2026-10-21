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

package orchestrator_adapters

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/wdk/events/events_provider_gochannel"
)

func newGoChannelBus(t testing.TB) *watermillEventBus {
	t.Helper()
	provider, err := events_provider_gochannel.NewGoChannelProvider(events_provider_gochannel.DefaultConfig())
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, provider.Start(ctx))
	bus := NewWatermillEventBus(provider.Publisher(), provider.Subscriber(), provider.Router())
	t.Cleanup(func() {
		cancel()
		_ = bus.Close(context.Background())
		_ = provider.Close()
	})
	watermill, ok := bus.(*watermillEventBus)
	require.True(t, ok)
	return watermill
}

func TestRawEventBus_DeliversThePayloadUnchanged(t *testing.T) {
	bus := newGoChannelBus(t)
	received := make(chan []byte, 1)
	require.NoError(t, bus.SubscribeRaw(t.Context(), "tasks.raw", func(_ context.Context, payload []byte) error {
		received <- append([]byte(nil), payload...)
		return nil
	}))

	ref := orchestrator_domain.TaskRef{ID: "0192-task", Attempt: 3, Priority: orchestrator_domain.PriorityHigh}
	require.NoError(t, bus.PublishRaw(t.Context(), "tasks.raw", orchestrator_domain.AppendTaskRef(nil, ref)))

	select {
	case payload := <-received:
		decoded, err := orchestrator_domain.DecodeTaskRef(payload)
		require.NoError(t, err)
		assert.Equal(t, ref, decoded)
	case <-time.After(5 * time.Second):
		t.Fatal("the raw message was not delivered")
	}
}

func TestRawEventBus_FailedHandlerGetsTheMessageAgain(t *testing.T) {
	bus := newGoChannelBus(t)
	var calls atomic.Int32
	done := make(chan struct{})
	require.NoError(t, bus.SubscribeRaw(t.Context(), "tasks.retry", func(context.Context, []byte) error {
		if calls.Add(1) == 1 {
			return errors.New("store unavailable")
		}
		close(done)
		return nil
	}))

	require.NoError(t, bus.PublishRaw(t.Context(), "tasks.retry", []byte{1}))
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a message whose handler failed was not delivered again")
	}
	assert.Equal(t, int32(2), calls.Load())
}

func TestRawEventBus_RefusesOversizePayloadsAndAClosedBus(t *testing.T) {
	bus := newGoChannelBus(t)
	err := bus.PublishRaw(t.Context(), "tasks.big", make([]byte, DefaultMaxEventPayloadBytes+1))
	require.ErrorIs(t, err, ErrEventPayloadTooLarge)

	require.NoError(t, bus.Close(context.Background()))
	require.ErrorIs(t, bus.PublishRaw(t.Context(), "tasks.big", []byte{1}), orchestrator_domain.ErrServiceClosed)
	require.ErrorIs(t, bus.SubscribeRaw(t.Context(), "tasks.big", func(context.Context, []byte) error { return nil }),
		orchestrator_domain.ErrServiceClosed)
}

func TestRawEventBus_MessageIDsAreUniqueAcrossBuses(t *testing.T) {
	first, second := newGoChannelBus(t), newGoChannelBus(t)
	require.NotEqual(t, first.rawIDPrefix, second.rawIDPrefix)

	seen := make(map[string]struct{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 250 {
				id := first.nextRawID()
				mu.Lock()
				seen[id] = struct{}{}
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	assert.Len(t, seen, 1000, "concurrent publishes never reuse an ID")
}

func TestRawEventBus_PublishSendsNoMetadataWithoutASpan(t *testing.T) {
	capture := &capturingPublisher{}
	bus, ok := NewWatermillEventBus(capture, &recordingSubscriber{}, nil).(*watermillEventBus)
	require.True(t, ok)
	require.NoError(t, bus.PublishRaw(t.Context(), "tasks.plain", []byte{1, 2}))

	require.Len(t, capture.messages, 1)
	assert.Empty(t, capture.messages[0].Metadata, "an untraced publish carries no metadata")
	assert.Equal(t, []byte{1, 2}, []byte(capture.messages[0].Payload))
}

type capturingPublisher struct {
	messages []*message.Message
}

func (p *capturingPublisher) Publish(_ string, messages ...*message.Message) error {
	p.messages = append(p.messages, messages...)
	return nil
}

func (*capturingPublisher) Close() error { return nil }
