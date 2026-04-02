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
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"piko.sh/piko/internal/json"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

const (
	// logKeyTopic is the logging field key for the event bus topic name.
	logKeyTopic = "topic"

	// logKeyMessageID is the log key for the unique ID of a Watermill message.
	logKeyMessageID = "messageID"

	// logKeyPayloadSize is the log field key for message payload size in bytes.
	logKeyPayloadSize = "payloadSize"

	// logKeyDurationMs is the logging key for operation duration in milliseconds.
	logKeyDurationMs = "duration_ms"

	// eventBusSubscriptionBufferSize is the buffer size for event bus subscription
	// channels.
	eventBusSubscriptionBufferSize = 128
)

// watermillEventBus adapts Watermill's Publisher and Subscriber interfaces
// to implement the EventBus interface.
//
// This adapter enables the orchestrator to use any Watermill-compatible
// pub/sub implementation (GoChannel, Redis, NATS, Kafka, etc.) whilst
// maintaining the existing EventBus contract.
//
// It supports two subscription modes:
// 1. Subscribe() - Channel-based, fire-and-forget (messages Acked immediately)
// 2. SubscribeWithHandler() - Handler-based with manual Ack/Nack control
type watermillEventBus struct {
	// publisher sends messages to topics.
	publisher message.Publisher

	// subscriber receives messages from topics.
	subscriber message.Subscriber

	// router manages message routing and subscription handling.
	router *message.Router

	// subscriptions maps topic names to active subscription records for cleanup.
	subscriptions map[string]*watermillSubscription

	// subscriptionsMutex guards access to the subscriptions map.
	subscriptionsMutex sync.RWMutex

	// isClosed indicates whether the event bus has been closed.
	isClosed bool

	// closeMutex guards access to isClosed during shutdown checks.
	closeMutex sync.RWMutex
}

// watermillSubscription holds an active subscription to a Watermill topic.
type watermillSubscription struct {
	// outputChan sends events to subscribers; closed when unsubscribing.
	outputChan chan orchestrator_domain.Event

	// cancelFunc cancels the subscription's context to stop message processing.
	cancelFunc context.CancelCauseFunc

	// topic is the Watermill subscription topic name.
	topic string
}

// Publish converts an orchestrator Event to a Watermill Message and publishes
// it.
//
// The context's trace information is propagated via message metadata to enable
// distributed tracing across the pub/sub boundary.
//
// Takes topic (string) which specifies the destination topic for the message.
// Takes event (orchestrator_domain.Event) which contains the event data to
// publish.
//
// Returns error when the bus is closed, message creation fails, or publishing
// to Watermill fails.
func (b *watermillEventBus) Publish(ctx context.Context, topic string, event orchestrator_domain.Event) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "WatermillEventBus.Publish",
		logger_domain.String(logKeyTopic, topic),
		logger_domain.String("eventType", string(event.Type)),
		logger_domain.Int(logKeyPayloadSize, len(event.Payload)),
	)
	defer span.End()

	startTime := time.Now()

	if err := b.checkNotClosed(ctx, span); err != nil {
		WatermillEventBusPublishErrorCount.Add(ctx, 1)
		return fmt.Errorf("checking event bus state before publish: %w", err)
	}

	wmMessage, err := b.createMessage(ctx, span, event)
	if err != nil {
		WatermillEventBusPublishErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating message for topic %q: %w", topic, err)
	}

	l.Trace("Publishing message to Watermill",
		logger_domain.String(logKeyMessageID, wmMessage.UUID),
		logger_domain.Int(logKeyPayloadSize, len(wmMessage.Payload)))

	if err := b.publisher.Publish(topic, wmMessage); err != nil {
		l.ReportError(span, err, "Failed to publish message")
		WatermillEventBusPublishErrorCount.Add(ctx, 1)
		return fmt.Errorf("publishing to watermill: %w", err)
	}

	duration := time.Since(startTime)
	WatermillEventBusPublishDuration.Record(ctx, float64(duration.Milliseconds()))
	WatermillEventBusPublishedEvents.Add(ctx, 1)

	l.Trace("Event published successfully",
		logger_domain.Int64(logKeyDurationMs, duration.Milliseconds()),
		logger_domain.String(logKeyMessageID, wmMessage.UUID))

	return nil
}

// Subscribe creates a subscription to the given topic and returns a channel
// of orchestrator events.
//
// The subscription supports wildcard patterns (e.g., "artefact.*") by
// converting them to the appropriate Watermill subscription pattern.
//
// The returned channel will be closed when:
//   - The provided context is cancelled
//   - The EventBus is closed
//   - An error occurs in the subscription
//
// Takes topic (string) which specifies the topic name or wildcard pattern.
//
// Returns <-chan orchestrator_domain.Event which yields events as they are
// published to the topic.
// Returns error when the event bus is closed.
func (b *watermillEventBus) Subscribe(ctx context.Context, topic string) (<-chan orchestrator_domain.Event, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "WatermillEventBus.Subscribe",
		logger_domain.String(logKeyTopic, topic),
	)
	defer span.End()

	startTime := time.Now()

	if err := b.checkNotClosed(ctx, span); err != nil {
		WatermillEventBusSubscribeErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("checking event bus state before subscribe: %w", err)
	}

	outputChan, subCtx := b.createSubscription(ctx, topic)

	l.Internal("Creating Watermill subscription handler")
	handlerName := fmt.Sprintf("eventbus_%s_%s", topic, watermill.NewShortUUID())

	b.router.AddConsumerHandler(
		handlerName,
		topic,
		b.subscriber,
		b.createChannelMessageHandler(subCtx, topic, outputChan),
	)

	b.monitorContextCancellation(ctx, topic)

	duration := time.Since(startTime)
	WatermillEventBusSubscribeDuration.Record(ctx, float64(duration.Milliseconds()))
	WatermillEventBusSubscriberCount.Add(ctx, 1)

	l.Internal("Subscription created",
		logger_domain.Int64(logKeyDurationMs, duration.Milliseconds()),
		logger_domain.Int("bufferSize", eventBusSubscriptionBufferSize))

	return outputChan, nil
}

// SubscribeWithHandler subscribes to a topic using direct subscription with
// proper Ack/Nack semantics for at-least-once delivery.
//
// Unlike the router-based Subscribe, this method creates its own message
// processing goroutine, allowing subscriptions to be established after the
// router has started running. This is essential for dynamic subscriptions
// during application lifecycle.
//
// Takes topic (string) which specifies the topic name to subscribe to.
// Takes handler (EventHandler) which processes each received message.
//
// Returns error when the event bus is closed or subscription fails.
//
// The handler function is called for each message. If the handler returns nil,
// the message is Acked. If the handler returns an error, the message is Nacked
// and will be redelivered depending on the underlying pub/sub implementation.
//
// This method is suitable for critical message processing where message loss
// is unacceptable, such as processing artefact events that trigger
// orchestrator tasks.
//
// The handler MUST be idempotent as messages may be delivered multiple times.
func (b *watermillEventBus) SubscribeWithHandler(ctx context.Context, topic string, handler orchestrator_domain.EventHandler) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "WatermillEventBus.SubscribeWithHandler",
		logger_domain.String(logKeyTopic, topic),
	)
	defer span.End()

	startTime := time.Now()

	if err := b.checkNotClosed(ctx, span); err != nil {
		WatermillEventBusSubscribeErrorCount.Add(ctx, 1)
		return fmt.Errorf("checking event bus state before handler subscribe: %w", err)
	}

	l.Internal("Creating direct subscription to topic")

	messages, err := b.subscriber.Subscribe(ctx, topic)
	if err != nil {
		l.ReportError(span, err, "Failed to subscribe to topic")
		WatermillEventBusSubscribeErrorCount.Add(ctx, 1)
		return fmt.Errorf("subscribing to topic %q: %w", topic, err)
	}

	l.Internal("Direct subscription established",
		logger_domain.String(logKeyTopic, topic))

	b.startMessageProcessor(ctx, topic, messages, handler)

	duration := time.Since(startTime)
	WatermillEventBusSubscribeDuration.Record(ctx, float64(duration.Milliseconds()))
	WatermillEventBusSubscriberCount.Add(ctx, 1)

	l.Internal("Handler subscription created",
		logger_domain.Int64(logKeyDurationMs, duration.Milliseconds()))

	return nil
}

// Router returns the underlying Watermill router for advanced use cases.
// Users can add their own handlers to this router.
//
// Returns *message.Router which provides access to the internal message router.
func (b *watermillEventBus) Router() *message.Router {
	return b.router
}

// Publisher returns the underlying Watermill publisher.
//
// Returns message.Publisher which provides access to the raw publisher.
func (b *watermillEventBus) Publisher() message.Publisher {
	return b.publisher
}

// Subscriber returns the underlying Watermill subscriber.
//
// Returns message.Subscriber which provides access to the message consumer.
func (b *watermillEventBus) Subscriber() message.Subscriber {
	return b.subscriber
}

// Close shuts down the EventBus, closing all active subscriptions and
// stopping the Watermill router.
//
// Takes ctx (context.Context) which carries logging context for the shutdown
// operation.
//
// Returns error when shutdown fails, or nil if already closed.
func (b *watermillEventBus) Close(ctx context.Context) error {
	ctx, cl := logger_domain.From(context.WithoutCancel(ctx), log)
	ctx, span, l := cl.Span(ctx, "WatermillEventBus.Close")
	defer span.End()

	startTime := time.Now()

	if !b.markAsClosed() {
		l.Internal("Event bus already closed")
		return nil
	}

	l.Internal("Closing Watermill event bus")

	subscriptionCount := b.closeAllSubscriptions(ctx)
	b.closeWatermillComponents(ctx, span)

	duration := time.Since(startTime)
	WatermillEventBusCloseDuration.Record(ctx, float64(duration.Milliseconds()))

	l.Internal("Watermill event bus closed",
		logger_domain.Int64(logKeyDurationMs, duration.Milliseconds()),
		logger_domain.Int("closedSubscriptions", subscriptionCount))

	return nil
}

// checkNotClosed verifies the event bus is not closed.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes span (trace.Span) which receives error reports when closed.
//
// Returns error when the event bus has been closed.
//
// Safe for concurrent use; uses a read lock to check the closed state.
func (b *watermillEventBus) checkNotClosed(ctx context.Context, span trace.Span) error {
	ctx, l := logger_domain.From(ctx, log)
	b.closeMutex.RLock()
	defer b.closeMutex.RUnlock()

	if b.isClosed {
		l.Warn("Attempted to operate on closed event bus")
		l.ReportError(span, orchestrator_domain.ErrServiceClosed, "Attempted to operate on closed event bus")
		return orchestrator_domain.ErrServiceClosed
	}
	return nil
}

// createMessage creates a Watermill message from an orchestrator event.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes span (trace.Span) which provides trace context for error reporting.
// Takes event (orchestrator_domain.Event) which is the event to convert.
//
// Returns *message.Message which contains the serialised event with trace
// metadata.
// Returns error when the event cannot be marshalled to JSON.
func (*watermillEventBus) createMessage(
	ctx context.Context,
	span trace.Span,
	event orchestrator_domain.Event,
) (*message.Message, error) {
	ctx, l := logger_domain.From(ctx, log)
	payload, err := json.Marshal(event)
	if err != nil {
		l.ReportError(span, err, "Failed to marshal event")
		return nil, fmt.Errorf("marshalling event: %w", err)
	}

	wmMessage := message.NewMessage(watermill.NewUUID(), payload)

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		wmMessage.Metadata.Set(k, v)
	}

	wmMessage.Metadata.Set("event_type", string(event.Type))
	wmMessage.Metadata.Set("published_at", time.Now().Format(time.RFC3339Nano))

	return wmMessage, nil
}

// createSubscription creates a new subscription with output channel and
// registers it.
//
// Takes ctx (context.Context) which is the parent context for the
// subscription's cancellable context.
// Takes topic (string) which specifies the event topic to subscribe to.
//
// Returns chan orchestrator_domain.Event which yields events for the topic.
// Returns context.Context which is cancelled when the subscription ends.
//
// Safe for concurrent use. Uses mutex to protect subscription registration.
func (b *watermillEventBus) createSubscription(ctx context.Context, topic string) (chan orchestrator_domain.Event, context.Context) {
	outputChan := make(chan orchestrator_domain.Event, eventBusSubscriptionBufferSize)
	subCtx, cancel := context.WithCancelCause(ctx)

	sub := &watermillSubscription{
		outputChan: outputChan,
		cancelFunc: cancel,
		topic:      topic,
	}

	b.subscriptionsMutex.Lock()
	b.subscriptions[topic] = sub
	b.subscriptionsMutex.Unlock()

	return outputChan, subCtx
}

// createChannelMessageHandler creates a message handler that sends events to
// a channel.
//
// Takes topic (string) which names the subscription topic for logging.
// Takes outputChan (chan orchestrator_domain.Event) which receives decoded
// events from messages.
//
// Returns func(*message.Message) error which handles incoming messages by
// decoding them into events and sending them to the output channel.
func (*watermillEventBus) createChannelMessageHandler(
	subCtx context.Context,
	topic string,
	outputChan chan orchestrator_domain.Event,
) func(*message.Message) error {
	return func(wmMessage *message.Message) error {
		carrier := propagation.MapCarrier{}
		maps.Copy(carrier, wmMessage.Metadata)
		msgCtx := otel.GetTextMapPropagator().Extract(subCtx, carrier)

		msgCtx, ml := logger_domain.From(msgCtx, log)
		msgCtx, msgSpan, msgLog := ml.Span(msgCtx, "WatermillEventBus.MessageHandler",
			logger_domain.String(logKeyTopic, topic),
			logger_domain.String(logKeyMessageID, wmMessage.UUID),
		)
		defer msgSpan.End()

		var event orchestrator_domain.Event
		if err := json.Unmarshal(wmMessage.Payload, &event); err != nil {
			msgLog.ReportError(msgSpan, err, "Failed to unmarshal event from message")
			WatermillEventBusMessageUnmarshalErrorCount.Add(msgCtx, 1)
			wmMessage.Ack()
			return nil
		}

		msgLog.Trace("Received event from Watermill",
			logger_domain.String("eventType", string(event.Type)),
			logger_domain.Int(logKeyPayloadSize, len(event.Payload)))

		select {
		case outputChan <- event:
			WatermillEventBusReceivedEvents.Add(msgCtx, 1)
			wmMessage.Ack()
		case <-subCtx.Done():
			msgLog.Trace("Subscription cancelled, not sending event")
			wmMessage.Nack()
		default:
			msgLog.Warn("Output channel full, dropping event")
			WatermillEventBusDroppedEvents.Add(msgCtx, 1)
			wmMessage.Nack()
		}

		return nil
	}
}

// monitorContextCancellation starts a background task to clean up
// the subscription on context cancellation.
//
// Takes ctx (context.Context) which is the subscription context to monitor.
// Takes topic (string) which identifies the subscription to clean
// up when the context is done.
//
// Concurrent goroutine is spawned that blocks until the context is
// cancelled, then cleans up the subscription.
func (b *watermillEventBus) monitorContextCancellation(ctx context.Context, topic string) {
	go func() {
		<-ctx.Done()

		_, bl := logger_domain.From(context.WithoutCancel(ctx), log)
		_, bgSpan, bgLog := bl.Span(context.WithoutCancel(ctx), "WatermillEventBus.ContextCancellation",
			logger_domain.String(logKeyTopic, topic),
			logger_domain.String("cancelReason", context.Cause(ctx).Error()),
		)
		defer bgSpan.End()

		bgLog.Trace("Context done, cleaning up subscription",
			logger_domain.String("cancelReason", context.Cause(ctx).Error()))

		b.unsubscribe(ctx, topic)
	}()
}

// startMessageProcessor starts a background task that processes
// messages from the channel until it is closed or the context
// is cancelled.
//
// Takes topic (string) which identifies the subscription topic for
// logging.
// Takes messages (<-chan *message.Message) which provides the
// Watermill messages to process.
// Takes handler (orchestrator_domain.EventHandler) which processes
// each decoded event.
//
// Concurrent goroutine is spawned that processes messages until the
// channel is closed or the context is cancelled.
func (b *watermillEventBus) startMessageProcessor(
	ctx context.Context,
	topic string,
	messages <-chan *message.Message,
	handler orchestrator_domain.EventHandler,
) {
	ctx, l := logger_domain.From(ctx, log)
	ready := make(chan struct{})
	go func() {
		defer goroutine.RecoverPanic(ctx, "orchestrator.messageProcessor")
		processorLog := l.With(
			logger_domain.String(logKeyTopic, topic),
			logger_domain.String("component", "SubscribeWithHandler.processor"),
		)
		processorLog.Internal("Message processor started")
		close(ready)

		for {
			select {
			case <-ctx.Done():
				processorLog.Trace("Context cancelled, stopping message processor")
				return
			case wmMessage, ok := <-messages:
				if !ok {
					processorLog.Trace("Message channel closed, stopping processor")
					return
				}
				if ctx.Err() != nil {
					wmMessage.Nack()
					return
				}
				b.processMessage(ctx, topic, wmMessage, handler)
			}
		}
	}()
	<-ready
}

// processMessage handles a single message from a subscription, calling the
// handler and managing Ack/Nack based on the result.
//
// Takes topic (string) which identifies the subscription topic.
// Takes wmMessage (*message.Message) which contains the message
// payload and metadata.
// Takes handler (orchestrator_domain.EventHandler) which processes the event.
func (*watermillEventBus) processMessage(
	ctx context.Context,
	topic string,
	wmMessage *message.Message,
	handler orchestrator_domain.EventHandler,
) {
	carrier := propagation.MapCarrier{}
	maps.Copy(carrier, wmMessage.Metadata)
	msgCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)

	msgCtx, msgL := logger_domain.From(msgCtx, log)
	msgCtx, msgSpan, msgLog := msgL.Span(msgCtx, "WatermillEventBus.ProcessMessage",
		logger_domain.String(logKeyTopic, topic),
		logger_domain.String(logKeyMessageID, wmMessage.UUID),
	)
	defer msgSpan.End()

	msgLog.Trace("Processing message",
		logger_domain.String(logKeyMessageID, wmMessage.UUID))

	var event orchestrator_domain.Event
	if err := json.Unmarshal(wmMessage.Payload, &event); err != nil {
		msgLog.ReportError(msgSpan, err, "Failed to unmarshal event from message")
		WatermillEventBusMessageUnmarshalErrorCount.Add(msgCtx, 1)
		wmMessage.Nack()
		return
	}

	msgLog.Trace("Calling event handler",
		logger_domain.String("eventType", string(event.Type)),
		logger_domain.Int(logKeyPayloadSize, len(event.Payload)))

	handlerStartTime := time.Now()

	if err := handler(msgCtx, event); err != nil {
		msgLog.ReportError(msgSpan, err, "Handler returned error, message Nacked")
		WatermillEventBusDroppedEvents.Add(msgCtx, 1)
		wmMessage.Nack()
		return
	}

	handlerDuration := time.Since(handlerStartTime)
	msgLog.Trace("Handler completed successfully",
		logger_domain.Int64(logKeyDurationMs, handlerDuration.Milliseconds()))

	WatermillEventBusReceivedEvents.Add(msgCtx, 1)
	wmMessage.Ack()
}

// markAsClosed atomically marks the bus as closed.
//
// Returns bool which is true if this call closed the bus, or false if it was
// already closed.
//
// Safe for concurrent use.
func (b *watermillEventBus) markAsClosed() bool {
	b.closeMutex.Lock()
	defer b.closeMutex.Unlock()

	if b.isClosed {
		return false
	}
	b.isClosed = true
	return true
}

// closeAllSubscriptions closes all active subscriptions and returns the count.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
//
// Returns int which is the number of subscriptions that were closed.
//
// Safe for concurrent use. Holds the subscription mutex while closing.
func (b *watermillEventBus) closeAllSubscriptions(ctx context.Context) int {
	ctx, l := logger_domain.From(ctx, log)
	b.subscriptionsMutex.Lock()
	subscriptionCount := len(b.subscriptions)
	for topic, sub := range b.subscriptions {
		l.Trace("Closing subscription", logger_domain.String(logKeyTopic, topic))
		sub.cancelFunc(errors.New("event bus closing all subscriptions"))
		close(sub.outputChan)
	}
	b.subscriptions = nil
	b.subscriptionsMutex.Unlock()

	if subscriptionCount > 0 {
		WatermillEventBusSubscriberCount.Add(ctx, -int64(subscriptionCount))
	}
	return subscriptionCount
}

// closeWatermillComponents closes the router, publisher, and subscriber.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes span (trace.Span) which records errors during component closure.
func (b *watermillEventBus) closeWatermillComponents(ctx context.Context, span trace.Span) {
	ctx, l := logger_domain.From(ctx, log)
	if err := b.router.Close(); err != nil {
		l.ReportError(span, err, "Error closing Watermill router")
		WatermillEventBusCloseErrorCount.Add(ctx, 1)
	}

	if err := b.publisher.Close(); err != nil {
		l.ReportError(span, err, "Error closing Watermill publisher")
		WatermillEventBusCloseErrorCount.Add(ctx, 1)
	}

	if err := b.subscriber.Close(); err != nil {
		l.ReportError(span, err, "Error closing Watermill subscriber")
		WatermillEventBusCloseErrorCount.Add(ctx, 1)
	}
}

// unsubscribe removes a subscription and cleans up its resources.
//
// Takes ctx (context.Context) which carries logging context for the
// unsubscribe operation.
// Takes topic (string) which identifies the subscription to remove.
//
// Safe for concurrent use; protects subscriptions with a mutex.
func (b *watermillEventBus) unsubscribe(ctx context.Context, topic string) {
	ctx, ul := logger_domain.From(context.WithoutCancel(ctx), log)
	ctx, span, l := ul.Span(ctx, "WatermillEventBus.unsubscribe",
		logger_domain.String(logKeyTopic, topic),
	)
	defer span.End()

	b.subscriptionsMutex.Lock()
	defer b.subscriptionsMutex.Unlock()

	sub, exists := b.subscriptions[topic]
	if !exists {
		l.Trace("Subscription not found")
		return
	}

	l.Internal("Unsubscribing from topic")

	sub.cancelFunc(errors.New("event bus subscription unsubscribed"))
	close(sub.outputChan)
	delete(b.subscriptions, topic)

	WatermillEventBusSubscriberCount.Add(ctx, -1)

	l.Internal("Unsubscribed from topic",
		logger_domain.Int("remainingSubscriptions", len(b.subscriptions)))
}

// NewWatermillEventBus creates a new EventBus adapter using the provided
// Watermill Publisher and Subscriber.
//
// The router manages subscription lifecycle and message routing. It will be
// started automatically when the first subscription is created.
//
// Takes publisher (message.Publisher) which handles sending messages.
// Takes subscriber (message.Subscriber) which handles receiving messages.
// Takes router (*message.Router) which manages subscription lifecycle.
//
// Returns orchestrator_domain.EventBus which is the configured event bus
// ready for use.
func NewWatermillEventBus(
	publisher message.Publisher,
	subscriber message.Subscriber,
	router *message.Router,
) orchestrator_domain.EventBus {
	return &watermillEventBus{
		publisher:          publisher,
		subscriber:         subscriber,
		router:             router,
		subscriptions:      make(map[string]*watermillSubscription),
		subscriptionsMutex: sync.RWMutex{},
		isClosed:           false,
		closeMutex:         sync.RWMutex{},
	}
}
