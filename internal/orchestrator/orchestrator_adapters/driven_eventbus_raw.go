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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// rawIDPrefixBytes is the number of random bytes in a raw message ID prefix.
	rawIDPrefixBytes = 8

	// rawHandlerRetryDelay is the delay before redelivering a failed raw message.
	rawHandlerRetryDelay = 100 * time.Millisecond

	// rawIDSequenceBase is the base a raw message ID's sequence number is written in.
	rawIDSequenceBase = 36
)

var (
	_ orchestrator_domain.RawEventBus = (*watermillEventBus)(nil)
)

// PublishRaw sends payload to topic as the whole message body, with no Event envelope,
// span or timestamp metadata. Trace context is attached only when ctx carries a span.
//
// Calls may run concurrently.
//
// Takes topic (string) which is the destination.
// Takes payload ([]byte) which the bus owns once called.
//
// Returns error when the bus is closed, the payload exceeds the cap
// (ErrEventPayloadTooLarge), or the publisher refuses the message.
func (b *watermillEventBus) PublishRaw(ctx context.Context, topic string, payload []byte) error {
	if b.closed() {
		WatermillEventBusPublishErrorCount.Add(ctx, 1)
		return orchestrator_domain.ErrServiceClosed
	}
	if maxPayloadBytes := b.effectiveMaxPayloadBytes(); int64(len(payload)) > maxPayloadBytes {
		WatermillEventBusPublishErrorCount.Add(ctx, 1)
		return fmt.Errorf("publishing to topic %q: payload size %d exceeds limit %d: %w",
			topic, len(payload), maxPayloadBytes, ErrEventPayloadTooLarge)
	}

	wmMessage := message.NewMessage(b.nextRawID(), payload)
	if trace.SpanContextFromContext(ctx).IsValid() {
		otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(wmMessage.Metadata))
	}
	if err := b.publisher.Publish(topic, wmMessage); err != nil {
		WatermillEventBusPublishErrorCount.Add(ctx, 1)
		return fmt.Errorf("publishing to watermill: %w", err)
	}
	WatermillEventBusPublishedEvents.Add(ctx, 1)
	return nil
}

// SubscribeRaw delivers each message on topic to handler from one processor goroutine.
// The subscriber shares a topic's messages between the processes subscribed to it.
//
// Takes topic (string) which is the topic.
// Takes handler (orchestrator_domain.RawHandler) which receives each payload; a message
// whose handler fails is returned for redelivery after a short delay.
//
// Returns error when the bus is closed or the subscription cannot be created.
func (b *watermillEventBus) SubscribeRaw(ctx context.Context, topic string, handler orchestrator_domain.RawHandler) error {
	if b.closed() {
		WatermillEventBusSubscribeErrorCount.Add(ctx, 1)
		return orchestrator_domain.ErrServiceClosed
	}
	messages, err := b.subscriber.Subscribe(ctx, topic)
	if err != nil {
		WatermillEventBusSubscribeErrorCount.Add(ctx, 1)
		return fmt.Errorf("subscribing to topic %q: %w", topic, err)
	}
	WatermillEventBusSubscriberCount.Add(ctx, 1)

	ready := make(chan struct{})
	b.processorWaitGroup.Go(func() {
		defer goroutine.RecoverPanic(ctx, "orchestrator.rawMessageProcessor")
		close(ready)
		maxPayloadBytes := b.effectiveMaxPayloadBytes()
		for {
			select {
			case <-ctx.Done():
				return
			case wmMessage, ok := <-messages:
				if !ok {
					return
				}
				if ctx.Err() != nil {
					wmMessage.Nack()
					return
				}
				processRawMessage(ctx, topic, wmMessage, handler, maxPayloadBytes)
			}
		}
	})
	<-ready
	return nil
}

// processRawMessage hands one raw message to its handler and acknowledges it, or returns
// it for redelivery when the handler fails.
//
// Takes topic (string) which names the topic for logging.
// Takes wmMessage (*message.Message) which is the message.
// Takes handler (orchestrator_domain.RawHandler) which receives the payload.
// Takes maxPayloadBytes (int64) which is the largest payload accepted; a larger one is
// acknowledged and dropped.
func processRawMessage(
	ctx context.Context,
	topic string,
	wmMessage *message.Message,
	handler orchestrator_domain.RawHandler,
	maxPayloadBytes int64,
) {
	msgCtx := ctx
	if len(wmMessage.Metadata) > 0 {
		msgCtx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(wmMessage.Metadata))
	}
	if int64(len(wmMessage.Payload)) > maxPayloadBytes {
		WatermillEventBusMessageUnmarshalErrorCount.Add(msgCtx, 1)
		wmMessage.Ack()
		return
	}
	if err := handler(msgCtx, wmMessage.Payload); err != nil {
		_, l := logger_domain.From(msgCtx, log)
		l.Warn("Raw message handler failed; returning the message for redelivery",
			logger_domain.String(logKeyTopic, topic),
			logger_domain.Error(err))
		WatermillEventBusDroppedEvents.Add(msgCtx, 1)
		timer := time.NewTimer(rawHandlerRetryDelay)
		select {
		case <-timer.C:
		case <-ctx.Done():
		}
		timer.Stop()
		wmMessage.Nack()
		return
	}
	WatermillEventBusReceivedEvents.Add(msgCtx, 1)
	wmMessage.Ack()
}

// closed reports whether the bus has been closed.
//
// Returns bool which is true once Close has begun.
//
// Safe for concurrent use.
func (b *watermillEventBus) closed() bool {
	b.closeMutex.RLock()
	defer b.closeMutex.RUnlock()
	return b.isClosed
}

// nextRawID combines the bus's random prefix and next sequence number into a message ID.
//
// An atomic sequence counter allows concurrent calls.
//
// Returns string which is the ID.
func (b *watermillEventBus) nextRawID() string {
	var buf [2*rawIDPrefixBytes + 1 + 13]byte
	id := append(buf[:0], b.rawIDPrefix...)
	id = append(id, '-')
	id = strconv.AppendUint(id, b.rawIDSequence.Add(1), rawIDSequenceBase)
	return string(id)
}

// newRawIDPrefix returns a random prefix for a bus's raw message IDs.
//
// Returns string which is the prefix in hexadecimal.
func newRawIDPrefix() string {
	var random [rawIDPrefixBytes]byte
	_, _ = rand.Read(random[:])
	return hex.EncodeToString(random[:])
}
