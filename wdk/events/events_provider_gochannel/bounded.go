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
	"strings"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	// messageBookkeepingBytes estimates retained overhead for each queued or delivering
	// message.
	messageBookkeepingBytes = 256

	// metadataEntryBytes estimates retained overhead for each message metadata pair.
	metadataEntryBytes = 32
)

var (
	// ErrPublishOverloaded indicates that no part of the publication was accepted.
	ErrPublishOverloaded = errors.New("event publication capacity exhausted")

	// ErrMessageTooLarge reports a batch larger than the whole publication byte budget. It
	// can never be admitted, however long the caller waits.
	ErrMessageTooLarge = errors.New("event publication larger than the publication budget")

	// ErrPublisherClosed indicates that publication has stopped permanently.
	ErrPublisherClosed = errors.New("event publisher closed")
)

// PublishOverloadError identifies the exhausted publication budget.
type PublishOverloadError struct {
	// Limit identifies the exhausted message, byte, or topic budget.
	Limit string
}

// Error describes the exhausted publication budget.
//
// Returns string which combines the overload error with the exhausted limit's name.
func (e *PublishOverloadError) Error() string {
	return fmt.Sprintf("%s: %s", ErrPublishOverloaded, e.Limit)
}

// Unwrap exposes the publication overload sentinel.
//
// Returns error which is ErrPublishOverloaded for errors.Is checks.
func (*PublishOverloadError) Unwrap() error { return ErrPublishOverloaded }

// Temporary reports that the overload clears as published messages are delivered.
//
// Returns bool which is always true.
func (*PublishOverloadError) Temporary() bool { return true }

// MessageTooLargeError reports a batch that exceeds the whole publication byte budget.
type MessageTooLargeError struct {
	// Size is the estimated retained size of the batch in bytes.
	Size int64

	// Limit is the publication byte budget.
	Limit int64
}

// Error describes the rejected batch.
//
// Returns string which names the batch size and the budget.
func (e *MessageTooLargeError) Error() string {
	return fmt.Sprintf("%s: %d bytes exceeds %d", ErrMessageTooLarge, e.Size, e.Limit)
}

// Unwrap returns ErrMessageTooLarge so callers can match it with errors.Is.
//
// Returns error which is ErrMessageTooLarge.
func (*MessageTooLargeError) Unwrap() error { return ErrMessageTooLarge }

// Permanent reports that retrying the same batch can never succeed.
//
// Returns bool which is always true.
func (*MessageTooLargeError) Permanent() bool { return true }

// PublicationStats counts queued and delivering messages together.
type PublicationStats struct {
	// Messages counts queued and currently delivering messages.
	Messages int

	// Bytes estimates retained message bytes, including payloads, metadata, and bookkeeping.
	Bytes int64

	// Topics counts active per-topic delivery workers.
	Topics int

	// Rejected counts publication batches rejected because an admission limit was exhausted.
	Rejected uint64
}

// publication holds one retained message and its optional completion signal.
type publication struct {
	// msg holds the copied message retained until delivery completes.
	msg *message.Message

	// done receives the delivery result when acknowledgement waiting is enabled.
	done chan error

	// bytes records the capacity charged to this message.
	bytes int64
}

// publicationCompletion pairs a delivery result with the caller cancellation context.
type publicationCompletion struct {
	// ctx carries the caller cancellation while waiting for acknowledgement.
	ctx context.Context

	// done receives the delivery result when acknowledgement waiting is enabled.
	done chan error
}

// publicationTopic holds the queue serviced by one topic delivery worker.
type publicationTopic struct {
	// queue contains accepted messages pending delivery; the publisher mutex guards access.
	queue []publication

	// held counts the topic's queued and delivering messages; the publisher mutex guards
	// access.
	held int

	// reserved records whether the topic may use the reserved capacity.
	reserved bool
}

// boundedPublisher owns one delivery worker per active topic. The wrapped publisher must
// wait for subscriber acknowledgements, bounding its internal goroutines too.
type boundedPublisher struct {
	// downstream delivers messages and waits for subscriber acknowledgements.
	downstream message.Publisher

	// closeErr stores the downstream close result after closeOnce completes.
	closeErr error

	// topics maps active topic names to their queues; mu guards access.
	topics map[string]*publicationTopic

	// capacityFreed receives a value, without blocking, whenever a delivered message
	// releases its capacity, so a publisher waiting for room can retry.
	capacityFreed chan struct{}

	// config holds immutable publication limits and acknowledgement settings.
	config Config

	// wg tracks delivery workers until shutdown completes.
	wg sync.WaitGroup

	// sharedMessages limits messages retained by topics outside the reserved prefixes.
	sharedMessages int

	// sharedBytes limits bytes retained by topics outside the reserved prefixes.
	sharedBytes int64

	// topicLimit caps the messages one topic may hold, from MaxTopicShare; zero means no
	// cap.
	topicLimit int

	// stats tracks retained work and rejected batches; mu guards access.
	stats PublicationStats

	// mu guards topics, stats, and closed.
	mu sync.Mutex

	// closeOnce runs downstream shutdown and worker draining once.
	closeOnce sync.Once

	// closed prevents new admission and causes queued deliveries to fail.
	closed bool
}

// newBoundedPublisher wraps a publisher with bounded per-topic delivery queues.
//
// Takes downstream (message.Publisher) which must wait for subscriber acknowledgements
// before returning from Publish.
// Takes config (Config) which supplies resolved capacity limits and acknowledgement
// settings.
//
// Returns *boundedPublisher which is ready to accept publications.
func newBoundedPublisher(downstream message.Publisher, config Config) *boundedPublisher {
	reserved := 0
	if len(config.ReservedTopicPrefixes) > 0 {
		reserved = min(max(config.ReservedCapacity, 0), config.PublishQueueCapacity)
	}
	sharedBytes := config.PublishQueueBytes
	if reserved > 0 {
		sharedBytes -= config.PublishQueueBytes * int64(reserved) / int64(config.PublishQueueCapacity)
	}
	topicLimit := 0
	if config.MaxTopicShare > 0 {
		topicLimit = max(int(float64(config.PublishQueueCapacity)*config.MaxTopicShare), 1)
	}
	return &boundedPublisher{
		downstream: downstream, config: config, topics: make(map[string]*publicationTopic),
		closeErr: nil, wg: sync.WaitGroup{}, stats: PublicationStats{}, mu: sync.Mutex{},
		closeOnce: sync.Once{}, closed: false, capacityFreed: make(chan struct{}, 1),
		sharedMessages: config.PublishQueueCapacity - reserved, sharedBytes: sharedBytes,
		topicLimit: topicLimit,
	}
}

// Publish admits a complete batch within the configured limits and queues it for
// delivery.
//
// When acknowledgement waiting is enabled, the call waits for delivery or caller
// cancellation. Accepted messages retain their capacity until delivery completes.
//
// Takes topic (string) which identifies the destination queue.
// Takes messages (...*message.Message) which contains the batch to validate and copy.
//
// Returns error when admission fails, or when acknowledgement waiting encounters delivery
// failure or cancellation.
//
// Safe for concurrent use; the publisher mutex guards admission and each active topic has
// one delivery goroutine.
func (p *boundedPublisher) Publish(topic string, messages ...*message.Message) error {
	if len(messages) == 0 {
		return nil
	}
	byteLimit := p.config.PublishQueueBytes
	if !p.isReservedTopic(topic) {
		byteLimit = p.sharedBytes
	}
	size, err := publicationSize(topic, messages, byteLimit)
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrPublisherClosed
	}
	t := p.topics[topic]
	if err == nil {
		err = p.checkCapacity(topic, t, len(messages), size)
	}
	if err != nil {
		if errors.Is(err, ErrPublishOverloaded) {
			p.stats.Rejected++
			publicationRejected.Add(context.Background(), 1)
		}
		p.mu.Unlock()
		return err
	}
	start := t == nil
	if start {
		t = &publicationTopic{queue: nil, held: 0, reserved: p.isReservedTopic(topic)}
		p.topics[topic] = t
		p.wg.Add(1)
		publicationChange(0, 0, 1)
	}
	completions := p.enqueue(topic, t, messages)
	p.stats.Messages += len(messages)
	p.stats.Bytes += size
	publicationChange(len(messages), size, 0)
	p.mu.Unlock()
	if start {
		go p.deliver(topic, t)
	}
	return awaitPublications(completions)
}

// Close stops admission, closes the downstream publisher, and waits for delivery workers.
//
// Returns error which is the downstream close result, shared by repeated calls.
//
// Safe for concurrent use; closeOnce serialises shutdown and the publisher mutex guards
// the closed flag.
func (p *boundedPublisher) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()

		p.closeErr = p.downstream.Close()
		p.wg.Wait()
	})
	return p.closeErr
}

// CapacitySignal signals released publication capacity.
//
// Signals may combine several releases, so receivers should publish as much as fits.
//
// Returns <-chan struct{} which signals released capacity.
func (p *boundedPublisher) CapacitySignal() <-chan struct{} {
	return p.capacityFreed
}

// FreeCapacity reports how many more messages a topic outside the reserved prefixes, such
// as a task dispatch topic, can admit now.
//
// Returns int which is the number of free message slots; a batch may still be refused for
// its bytes, its topic share or for starting a new topic.
//
// Safe for concurrent use; holds the publisher mutex while reading the counters.
func (p *boundedPublisher) FreeCapacity() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0
	}
	return max(p.sharedMessages-p.stats.Messages, 0)
}

// isReservedTopic reports whether topic may use the reserved capacity.
//
// Takes topic (string) which is the topic to classify.
//
// Returns bool which is true when the topic starts with a reserved prefix.
func (p *boundedPublisher) isReservedTopic(topic string) bool {
	for _, prefix := range p.config.ReservedTopicPrefixes {
		if strings.HasPrefix(topic, prefix) {
			return true
		}
	}
	return false
}

// checkCapacity checks whether the complete batch fits the remaining publication budgets.
// A topic outside the reserved prefixes may not use the reserved capacity, and no topic
// may hold more than its MaxTopicShare.
//
// Takes topic (string) which identifies the destination.
// Takes t (*publicationTopic) which is the topic's active queue, or nil when the topic
// needs a new worker.
// Takes count (int) which is the number of messages to admit.
// Takes size (int64) which is the estimated retained byte cost of the batch.
//
// Returns error which identifies the exhausted limit, or nil when the batch fits.
//
// The caller must hold the publisher mutex.
func (p *boundedPublisher) checkCapacity(topic string, t *publicationTopic, count int, size int64) error {
	var held int
	var reserved bool
	if t != nil {
		held, reserved = t.held, t.reserved
	} else {
		reserved = p.isReservedTopic(topic)
	}
	messageLimit, byteLimit := p.config.PublishQueueCapacity, p.config.PublishQueueBytes
	if !reserved {
		messageLimit, byteLimit = p.sharedMessages, p.sharedBytes
	}
	switch {
	case count > messageLimit-p.stats.Messages:
		return &PublishOverloadError{Limit: "messages"}
	case size > byteLimit-p.stats.Bytes:
		return &PublishOverloadError{Limit: "bytes"}
	case p.topicLimit > 0 && count > p.topicLimit-held:
		return &PublishOverloadError{Limit: "topic share"}
	case t == nil && len(p.topics) >= p.config.MaxPublishTopics:
		return &PublishOverloadError{Limit: "topics"}
	default:
		return nil
	}
}

// enqueue copies accepted messages into the topic queue and prepares acknowledgement
// waits.
//
// Takes topic (string) which identifies the destination for message size accounting.
// Takes t (*publicationTopic) which receives the copied messages.
// Takes messages ([]*message.Message) which contains the validated batch to enqueue.
//
// Returns []publicationCompletion which contains delivery waits, or nil when
// acknowledgement waiting is disabled.
//
// The caller must hold the publisher mutex and reserve capacity for the complete batch.
func (p *boundedPublisher) enqueue(topic string, t *publicationTopic, messages []*message.Message) []publicationCompletion {
	var completions []publicationCompletion
	if p.config.BlockPublishUntilSubscriberAck {
		completions = make([]publicationCompletion, 0, len(messages))
	}
	for _, msg := range messages {
		copied := msg.Copy()
		copied.SetContext(msg.Context())
		copied.Payload = append(message.Payload(nil), msg.Payload...)
		entry := publication{msg: copied, bytes: messageWeight(topic, msg), done: nil}
		if p.config.BlockPublishUntilSubscriberAck {
			entry.done = make(chan error, 1)
			completions = append(completions, publicationCompletion{ctx: msg.Context(), done: entry.done})
		}
		t.queue = append(t.queue, entry)
	}
	t.held += len(messages)
	return completions
}

// deliver drains a topic queue and releases each message's retained capacity.
//
// Takes topic (string) which identifies the downstream destination and active worker.
// Takes t (*publicationTopic) which contains the messages assigned to this worker.
//
// Concurrency: one goroutine drains each active topic; the publisher mutex protects queue
// and accounting updates while downstream delivery occurs outside the lock.
func (p *boundedPublisher) deliver(topic string, t *publicationTopic) {
	defer p.wg.Done()
	for {
		p.mu.Lock()
		if len(t.queue) == 0 {
			delete(p.topics, topic)
			publicationChange(0, 0, -1)
			p.mu.Unlock()
			return
		}
		item := t.queue[0]
		t.queue[0] = publication{}
		t.queue = t.queue[1:]
		closed := p.closed
		p.mu.Unlock()
		var err error
		if closed {
			err = ErrPublisherClosed
		} else {
			err = p.downstream.Publish(topic, item.msg)
		}
		p.mu.Lock()
		t.held--
		p.stats.Messages--
		p.stats.Bytes -= item.bytes
		publicationChange(-1, -item.bytes, 0)
		p.mu.Unlock()
		select {
		case p.capacityFreed <- struct{}{}:
		default:
		}
		if item.done != nil {
			item.done <- err
		}
	}
}

// snapshot reads publication counts and the number of active topic workers.
//
// Returns PublicationStats which contains a consistent copy of current accounting.
//
// Safe for concurrent use; holds the publisher mutex while reading counters and topics.
func (p *boundedPublisher) snapshot() PublicationStats {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := p.stats
	s.Topics = len(p.topics)
	return s
}

// publicationSize validates messages and sums their retained byte cost up to the limit.
//
// Takes topic (string) which contributes its name length to each message's cost.
// Takes messages ([]*message.Message) which contains the batch to validate and measure.
// Takes limit (int64) which is the maximum estimated byte cost allowed for the batch.
//
// Returns int64 which is the estimated retained byte cost, or zero on failure.
// Returns error when a message is nil, its context is cancelled, or the byte limit is
// exceeded.
func publicationSize(topic string, messages []*message.Message, limit int64) (int64, error) {
	var size int64
	for _, msg := range messages {
		if msg == nil {
			return 0, errors.New("nil event message")
		}
		if err := msg.Context().Err(); err != nil {
			return 0, err
		}
		size += messageWeight(topic, msg)
		if size > limit {
			return 0, &MessageTooLargeError{Size: size, Limit: limit}
		}
	}
	return size, nil
}

// messageWeight estimates the retained byte cost of one publication.
//
// Takes topic (string) which contributes its name length to the estimate.
// Takes msg (*message.Message) which supplies the payload, identifier, and metadata.
//
// Returns int64 which includes message data and bookkeeping overhead.
func messageWeight(topic string, msg *message.Message) int64 {
	weight := int64(len(msg.Payload) + len(msg.UUID) + len(topic) + messageBookkeepingBytes)
	for key, value := range msg.Metadata {
		weight += int64(len(key) + len(value) + metadataEntryBytes)
	}
	return weight
}

// awaitPublications waits for delivery results until all complete or one fails.
//
// Takes completions ([]publicationCompletion) which pairs each result channel with its
// cancellation context.
//
// Returns error when a delivery fails or a waiting caller cancels, or nil when all waits
// succeed.
func awaitPublications(completions []publicationCompletion) error {
	for _, completion := range completions {
		select {
		case err := <-completion.done:
			if err != nil {
				return err
			}
		case <-completion.ctx.Done():
			return completion.ctx.Err()
		}
	}
	return nil
}
