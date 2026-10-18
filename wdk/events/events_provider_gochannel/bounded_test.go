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

package events_provider_gochannel

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"
)

type blockedPublisher struct {
	stop chan struct{}
	once sync.Once
}

func (p *blockedPublisher) Publish(string, ...*message.Message) error { <-p.stop; return nil }
func (p *blockedPublisher) Close() error                              { p.once.Do(func() { close(p.stop) }); return nil }
func TestBoundedPublicationRejectsWholeBatchAndCloses(t *testing.T) {
	p := newBoundedPublisher(&blockedPublisher{stop: make(chan struct{})}, Config{PublishQueueCapacity: 2, PublishQueueBytes: 4096, MaxPublishTopics: 1})
	defer p.Close()
	require.NoError(t, p.Publish("blocked", message.NewMessage("1", nil)))
	require.ErrorIs(t, p.Publish("blocked", message.NewMessage("2", nil), message.NewMessage("3", nil)), ErrPublishOverloaded)
	require.Equal(t, 1, p.snapshot().Messages)
	require.ErrorIs(t, p.Publish("another", message.NewMessage("4", nil)), ErrPublishOverloaded)
	require.NoError(t, p.Close())
	require.Zero(t, p.snapshot().Messages)
	require.Zero(t, p.snapshot().Bytes)
	require.Zero(t, p.snapshot().Topics)
	require.ErrorIs(t, p.Publish("blocked", message.NewMessage("5", nil)), ErrPublisherClosed)
}
func TestBoundedGoChannelWaitsForAckWithoutGrowing(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PublishQueueCapacity = 4
	p, err := NewGoChannelProvider(cfg)
	require.NoError(t, err)
	defer p.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	sub, err := p.Subscriber().Subscribe(ctx, "slow")
	require.NoError(t, err)
	for i := range 4 {
		require.NoError(t, p.Publisher().Publish("slow", message.NewMessage(fmt.Sprint(i), nil)))
	}
	for range 10000 {
		require.True(t, errors.Is(p.Publisher().Publish("slow", message.NewMessage("excess", nil)), ErrPublishOverloaded))
	}
	require.Equal(t, 4, p.PublicationStats().Messages)
	for range 4 {
		select {
		case m := <-sub:
			m.Ack()
		case <-time.After(time.Second):
			t.Fatal("delivery stalled")
		}
	}
	require.Eventually(t, func() bool { return p.PublicationStats().Messages == 0 }, time.Second, time.Millisecond)
}
func TestBoundedPublicationCancellationAndSize(t *testing.T) {
	p, err := NewGoChannelProvider(DefaultConfig())
	require.NoError(t, err)
	defer p.Close()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	m := message.NewMessage("cancel", nil)
	m.SetContext(ctx)
	require.ErrorIs(t, p.Publisher().Publish("topic", m), context.Canceled)
	require.ErrorIs(t, p.Publisher().Publish("topic", message.NewMessage("large", make([]byte, 17<<20))), ErrMessageTooLarge)
	require.Zero(t, p.PublicationStats().Messages)
}

func TestSynchronousPublicationWaitsForAckAndCancellation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BlockPublishUntilSubscriberAck = true
	p, err := NewGoChannelProvider(cfg)
	require.NoError(t, err)
	defer p.Close()
	sub, err := p.Subscriber().Subscribe(t.Context(), "sync")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	m := message.NewMessage("one", nil)
	m.SetContext(ctx)
	done := make(chan error, 1)
	go func() { done <- p.Publisher().Publish("sync", m) }()
	received := <-sub
	select {
	case <-done:
		t.Fatal("returned before acknowledgement")
	default:
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	received.Ack()
	require.Eventually(t, func() bool { return p.PublicationStats().Messages == 0 }, time.Second, time.Millisecond)
}
func TestBlockedTopicDoesNotBlockOtherTopics(t *testing.T) {
	p, err := NewGoChannelProvider(DefaultConfig())
	require.NoError(t, err)
	defer p.Close()
	blocked, err := p.Subscriber().Subscribe(t.Context(), "blocked")
	require.NoError(t, err)
	fast, err := p.Subscriber().Subscribe(t.Context(), "fast")
	require.NoError(t, err)
	require.NoError(t, p.Publisher().Publish("blocked", message.NewMessage("one", nil)))
	held := <-blocked
	require.NoError(t, p.Publisher().Publish("fast", message.NewMessage("two", nil)))
	select {
	case m := <-fast:
		m.Ack()
	case <-time.After(time.Second):
		t.Fatal("unrelated topic stalled")
	}
	held.Ack()
}
func TestPublicationCopiesMutableMessageData(t *testing.T) {
	p, err := NewGoChannelProvider(DefaultConfig())
	require.NoError(t, err)
	defer p.Close()
	sub, err := p.Subscriber().Subscribe(t.Context(), "copy")
	require.NoError(t, err)
	m := message.NewMessage("one", []byte("original"))
	m.Metadata.Set("key", "original")
	require.NoError(t, p.Publisher().Publish("copy", m))
	m.Payload[0] = 'X'
	m.Metadata.Set("key", "changed")
	got := <-sub
	require.Equal(t, "original", string(got.Payload))
	require.Equal(t, "original", got.Metadata.Get("key"))
	got.Ack()
}

func TestClosedProviderCannotRestart(t *testing.T) {
	p, err := NewGoChannelProvider(DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, p.Close())
	require.ErrorIs(t, p.Start(t.Context()), ErrPublisherClosed)
}

func TestCloseReleasesRealSubscriberWithoutAcknowledgement(t *testing.T) {
	p, err := NewGoChannelProvider(DefaultConfig())
	require.NoError(t, err)
	sub, err := p.Subscriber().Subscribe(context.Background(), "blocked-close")
	require.NoError(t, err)
	for range 4 {
		require.NoError(t, p.Publisher().Publish("blocked-close", message.NewMessage("one", nil)))
	}
	<-sub
	done := make(chan error, 1)
	go func() { done <- p.Close() }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("close remained blocked on subscriber acknowledgement")
	}
	require.Zero(t, p.PublicationStats().Messages)
	require.Zero(t, p.PublicationStats().Bytes)
	require.Zero(t, p.PublicationStats().Topics)
}

func TestBoundedPublicationSignalsFreedCapacity(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PublishQueueCapacity = 2
	p, err := NewGoChannelProvider(cfg)
	require.NoError(t, err)
	defer p.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	sub, err := p.Subscriber().Subscribe(ctx, "signalled")
	require.NoError(t, err)

	bounded := p.bounded
	require.Equal(t, 2, bounded.FreeCapacity())
	require.NoError(t, p.Publisher().Publish("signalled", message.NewMessage("1", nil)))
	require.NoError(t, p.Publisher().Publish("signalled", message.NewMessage("2", nil)))
	require.Zero(t, bounded.FreeCapacity())
	require.ErrorIs(t, p.Publisher().Publish("signalled", message.NewMessage("3", nil)), ErrPublishOverloaded)

	(<-sub).Ack()
	select {
	case <-bounded.CapacitySignal():
	case <-time.After(time.Second):
		t.Fatal("freed capacity was not signalled")
	}
	require.Eventually(t, func() bool { return bounded.FreeCapacity() == 1 }, time.Second, time.Millisecond)
	(<-sub).Ack()
}

func TestBoundedPublicationTooLargeIsPermanent(t *testing.T) {
	p := newBoundedPublisher(&blockedPublisher{stop: make(chan struct{})},
		Config{PublishQueueCapacity: 4, PublishQueueBytes: 512, MaxPublishTopics: 1})
	defer p.Close()

	err := p.Publish("huge", message.NewMessage("1", make([]byte, 4096)))
	require.ErrorIs(t, err, ErrMessageTooLarge)
	require.False(t, errors.Is(err, ErrPublishOverloaded), "a message that can never fit is not a transient overload")
	var permanent interface{ Permanent() bool }
	require.True(t, errors.As(err, &permanent) && permanent.Permanent())
}

func TestBoundedPublicationReservesCapacityForReservedTopics(t *testing.T) {
	p := newBoundedPublisher(&blockedPublisher{stop: make(chan struct{})}, Config{
		PublishQueueCapacity: 4, PublishQueueBytes: 1 << 20, MaxPublishTopics: 4,
		ReservedTopicPrefixes: []string{"artefact."}, ReservedCapacity: 2,
	})
	defer p.Close()

	require.Equal(t, 2, p.FreeCapacity(), "the reserve is not free for other topics")
	require.NoError(t, p.Publish("task.dispatch", message.NewMessage("1", nil), message.NewMessage("2", nil)))
	err := p.Publish("task.dispatch", message.NewMessage("3", nil))
	require.ErrorIs(t, err, ErrPublishOverloaded, "a flood on other topics stops at the reserve")
	require.Zero(t, p.FreeCapacity())

	require.NoError(t, p.Publish("artefact.updated", message.NewMessage("4", nil), message.NewMessage("5", nil)),
		"artefact events still get in")
	require.ErrorIs(t, p.Publish("artefact.updated", message.NewMessage("6", nil)), ErrPublishOverloaded)
}

func TestBoundedPublicationReserveKeepsBytesBack(t *testing.T) {
	p := newBoundedPublisher(&blockedPublisher{stop: make(chan struct{})}, Config{
		PublishQueueCapacity: 4, PublishQueueBytes: 4096, MaxPublishTopics: 4,
		ReservedTopicPrefixes: []string{"artefact."}, ReservedCapacity: 2,
	})
	defer p.Close()

	err := p.Publish("task.dispatch", message.NewMessage("1", make([]byte, 3000)))
	require.ErrorIs(t, err, ErrMessageTooLarge, "a message beyond the shared byte budget can never fit")
	require.NoError(t, p.Publish("artefact.updated", message.NewMessage("2", make([]byte, 3000))))
}

func TestBoundedPublicationCapsTopicShare(t *testing.T) {
	p := newBoundedPublisher(&blockedPublisher{stop: make(chan struct{})}, Config{
		PublishQueueCapacity: 4, PublishQueueBytes: 1 << 20, MaxPublishTopics: 4,
		ReservedTopicPrefixes: []string{}, MaxTopicShare: 0.5,
	})
	defer p.Close()

	require.NoError(t, p.Publish("noisy", message.NewMessage("1", nil), message.NewMessage("2", nil)))
	err := p.Publish("noisy", message.NewMessage("3", nil))
	var overload *PublishOverloadError
	require.ErrorAs(t, err, &overload)
	require.Equal(t, "topic share", overload.Limit)
	require.NoError(t, p.Publish("quiet", message.NewMessage("4", nil), message.NewMessage("5", nil)),
		"one topic cannot take the whole queue")
}

func TestResolvePublicationLimits(t *testing.T) {
	resolved, err := resolvePublicationLimits(Config{})
	require.NoError(t, err)
	require.Equal(t, DefaultReservedTopicPrefixes(), resolved.ReservedTopicPrefixes)
	require.Equal(t, defaultPublishQueueCapacity/defaultReservedCapacityDivisor, resolved.ReservedCapacity)

	resolved, err = resolvePublicationLimits(Config{ReservedTopicPrefixes: []string{}})
	require.NoError(t, err)
	require.Empty(t, resolved.ReservedTopicPrefixes, "an empty, non-nil list reserves nothing")

	for name, config := range map[string]Config{
		"negative reserve":        {ReservedCapacity: -1},
		"reserve fills queue":     {PublishQueueCapacity: 8, ReservedCapacity: 8},
		"share above one":         {MaxTopicShare: 1.5},
		"negative share":          {MaxTopicShare: -0.1},
		"negative queue capacity": {PublishQueueCapacity: -1},
	} {
		_, err := resolvePublicationLimits(config)
		require.Error(t, err, name)
	}
}
