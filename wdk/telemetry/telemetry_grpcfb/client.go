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

package telemetry_grpcfb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sony/gobreaker/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"piko.sh/piko/internal/logger/logger_domain"
)

// Client is a reusable, non-blocking streaming telemetry shipper.
type Client struct {
	// currentBatch is the partially-filled batch accumulating appended events, nil when
	// empty.
	currentBatch *Batch

	// breaker guards the send path, opening after repeated stream failures.
	breaker *gobreaker.CircuitBreaker[struct{}]

	// cancel cancels the stream context derived in Start; Close calls it on teardown.
	cancel context.CancelCauseFunc

	// ticker drives the periodic flush of partially-filled batches.
	ticker *time.Ticker

	// stop signals the background goroutines to exit; closed by Close.
	stop chan struct{}

	// sendCh is the bounded queue of sealed batches the sender drains over the stream.
	sendCh chan *Batch

	// clientConnection is the underlying gRPC connection, closed on Close only when
	// ownsConnection.
	clientConnection *grpc.ClientConn

	// config is the effective configuration after defaults are applied.
	config Config

	// wg tracks the sender and flusher goroutines for Close to await.
	wg sync.WaitGroup

	// seq is the monotonically increasing sequence number stamped on sealed batches.
	seq int64

	// currentEventCount is the event count of the current batch, used to trigger a
	// size-based flush.
	currentEventCount int

	// sent counts events the server acknowledged ingesting.
	sent atomic.Int64

	// dropped counts events lost to a full queue, send failure, close or ack shortfall.
	dropped atomic.Int64

	// rejected counts events the server explicitly refused.
	rejected atomic.Int64

	// mu guards currentBatch, currentEventCount, seq and the send-queue close state.
	mu sync.Mutex

	// started reports whether Start has launched the background goroutines.
	started atomic.Bool

	// closed reports whether Close has begun, after which Add calls are ignored.
	closed atomic.Bool

	// dropWarned ensures the queue-full warning is logged at most once.
	dropWarned atomic.Bool

	// ownsConnection reports whether the Client created clientConnection and must close it
	// on Close.
	ownsConnection bool

	// sendClosed reports whether sendCh has been closed, guarded by mu.
	sendClosed bool
}

// preEncodedBatch carries an already-marshalled frame so the sender can reuse the bytes
// for SendMsg without marshalling twice. It is send-only.
//
// Carrying the pre-encoded bytes lets the sender size-check a batch (and reject an
// oversized one) before buffering it onto the shared stream.
type preEncodedBatch struct {
	// data is the marshalled batch frame handed straight to SendMsg.
	data []byte
}

// Config configures a Client. Zero values fall back to sensible defaults.
type Config struct {
	// Breaker tunes the send-path circuit breaker; nil uses the package defaults.
	Breaker *BreakerConfig

	// SiteID identifies the originating site, stamped on every batch.
	SiteID string

	// APIKey is the delivery key authenticating the stream, stamped on every batch.
	APIKey string

	// Source labels the producing component, stamped on every batch.
	Source string

	// Identity describes the emitting process and is stamped on every batch.
	Identity Identity

	// FlushSize is the event count that triggers a size-based flush; 0 uses the default.
	FlushSize int

	// FlushInterval is the period between time-based flushes; 0 uses the default.
	FlushInterval time.Duration

	// MaxQueuedBatches bounds the send queue; 0 uses the default.
	MaxQueuedBatches int
}

// Identity describes the process producing a telemetry stream.
type Identity struct {
	// InstanceID identifies this process among sibling replicas of the same service, stable
	// for the process lifetime.
	InstanceID string

	// Hostname is the emitting machine's hostname.
	Hostname string

	// ServiceName is the deployed service's name.
	ServiceName string

	// ServiceVersion is the running build's version.
	ServiceVersion string

	// Environment is the deployment environment ("production", "staging").
	Environment string

	// Region is the SERVICE's cloud region, not the user's: a user region needs licensed
	// GeoIP and cannot be derived here.
	Region string

	// StartedAtMs is when the process started, in epoch milliseconds.
	StartedAtMs int64

	// PID is the operating-system process identifier.
	PID int32
}

// BreakerConfig tunes the send-path circuit breaker.
type BreakerConfig struct {
	// MaxConsecutiveFailures is the failure count that trips the breaker open.
	MaxConsecutiveFailures int

	// OpenTimeout is how long the breaker stays open before probing again.
	OpenTimeout time.Duration
}

const (
	// defaultFlushSize is the event count that triggers a flush when FlushSize is unset.
	defaultFlushSize = 256

	// defaultFlushInterval is the flush period used when FlushInterval is unset.
	defaultFlushInterval = 5 * time.Second

	// defaultMaxQueuedBatches bounds the send queue when MaxQueuedBatches is unset.
	defaultMaxQueuedBatches = 64

	// defaultMaxFailures is the consecutive-failure count that trips the breaker by default.
	defaultMaxFailures = 5

	// defaultOpenTimeout is how long the breaker stays open by default.
	defaultOpenTimeout = 10 * time.Second

	// closeGracePeriod bounds how long Close waits, after cancelling the stream context, for
	// the sender goroutine to return from any in-flight stream call before the connection is
	// closed underneath it.
	closeGracePeriod = 2 * time.Second

	// logKeyEvents is the structured-log attribute key for an event count, reused across the
	// reconcile and send-path warnings.
	logKeyEvents = "events"
)

// withDefaults returns a copy of the config with unset fields replaced by their package
// defaults.
//
// Returns Config which is the config with FlushSize, FlushInterval and MaxQueuedBatches
// filled in where they were non-positive.
func (c *Config) withDefaults() Config {
	out := *c
	if out.FlushSize <= 0 {
		out.FlushSize = defaultFlushSize
	}
	if out.FlushInterval <= 0 {
		out.FlushInterval = defaultFlushInterval
	}
	if out.MaxQueuedBatches <= 0 {
		out.MaxQueuedBatches = defaultMaxQueuedBatches
	}
	return out
}

var (
	// ErrEmptyTarget is returned by Dial when the supplied target is empty.
	ErrEmptyTarget = errors.New("telemetry_grpcfb: empty dial target")

	// errClientClosed is the cancellation cause for the stream context when Close tears the
	// background goroutines down, so a cancelled in-flight stream reports why.
	errClientClosed = errors.New("telemetry_grpcfb: client closed")
)

// Marshal returns the pre-encoded frame bytes unchanged, satisfying the codec's encode
// side without re-marshalling.
//
// Returns []byte which is the already-encoded batch frame.
// Returns error which is always nil.
func (p preEncodedBatch) Marshal() ([]byte, error) { return p.data, nil }

// Unmarshal always fails because preEncodedBatch is send-only.
//
// Takes a []byte which is the wire payload, ignored.
//
// Returns error which always reports that the type cannot be decoded.
func (preEncodedBatch) Unmarshal([]byte) error {
	return errors.New("telemetry_grpcfb: preEncodedBatch is send-only")
}

// Start launches the background streamer and the periodic flush loop.
//
// The stream's cancellable context is derived from ctx here and passed to the goroutines
// as a parameter, so the Client retains no context.Context field. Start is idempotent, so
// subsequent calls are no-ops.
func (c *Client) Start(ctx context.Context) {
	if !c.started.CompareAndSwap(false, true) {
		return
	}
	streamCtx, cancel := context.WithCancelCause(ctx)
	c.cancel = cancel
	c.ticker = time.NewTicker(c.config.FlushInterval)
	c.wg.Go(func() { c.runSender(streamCtx) })
	c.wg.Go(func() { c.runFlusher(streamCtx) })
}

// AddAnalytics enqueues an analytics event (non-blocking).
//
// Takes e (AnalyticsEvent) which is the analytics event to enqueue.
//
//nolint:gocritic // hugeParam: a by-value event is the idiomatic public Add API.
func (c *Client) AddAnalytics(ctx context.Context, e AnalyticsEvent) {
	c.add(ctx, func(batch *Batch) { batch.Analytics = append(batch.Analytics, e) })
}

// AddWatchdog enqueues a watchdog event (non-blocking).
//
// Takes e (WatchdogEvent) which is the watchdog event to enqueue.
func (c *Client) AddWatchdog(ctx context.Context, e WatchdogEvent) {
	c.add(ctx, func(batch *Batch) { batch.Watchdog = append(batch.Watchdog, e) })
}

// AddLog enqueues a log line (non-blocking).
//
// Takes e (LogLine) which is the log line to enqueue.
func (c *Client) AddLog(ctx context.Context, e LogLine) {
	c.add(ctx, func(batch *Batch) { batch.Logs = append(batch.Logs, e) })
}

// AddSpan enqueues a trace span (non-blocking).
//
// Takes e (Span) which is the trace span to enqueue.
func (c *Client) AddSpan(ctx context.Context, e Span) {
	c.add(ctx, func(batch *Batch) { batch.Spans = append(batch.Spans, e) })
}

// AddMetric enqueues a metric point (non-blocking).
//
// Takes e (MetricPoint) which is the metric point to enqueue.
func (c *Client) AddMetric(ctx context.Context, e MetricPoint) {
	c.add(ctx, func(batch *Batch) { batch.Metrics = append(batch.Metrics, e) })
}

// AddError enqueues an error occurrence (non-blocking).
//
// Takes e (ErrorEvent) which is the error occurrence to enqueue.
func (c *Client) AddError(ctx context.Context, e ErrorEvent) {
	c.add(ctx, func(batch *Batch) { batch.Errors = append(batch.Errors, e) })
}

// AddProfile enqueues a captured profile (non-blocking).
//
// The compressed pprof bytes ride inline in ProfileMeta.Blob (bounded by the frame cap);
// when the bytes are stored out-of-band instead, ProfileMeta.BlobRef points at them and
// Blob is empty.
//
// Takes e (ProfileMeta) which is the captured profile to enqueue.
func (c *Client) AddProfile(ctx context.Context, e ProfileMeta) {
	c.add(ctx, func(batch *Batch) { batch.Profiles = append(batch.Profiles, e) })
}

// AddWorkerEvent enqueues a worker or job run-telemetry record (non-blocking).
//
// Takes e (WorkerEvent) which is the worker run-telemetry record to enqueue.
//
//nolint:gocritic // hugeParam: a by-value event is the idiomatic public Add API.
func (c *Client) AddWorkerEvent(ctx context.Context, e WorkerEvent) {
	c.add(ctx, func(batch *Batch) { batch.Workers = append(batch.Workers, e) })
}

// AddQueryStat enqueues a database query observation (non-blocking).
//
// Takes e (QueryStat) which is the database query observation to enqueue.
//
//nolint:gocritic // hugeParam: a by-value event is the idiomatic public Add API.
func (c *Client) AddQueryStat(ctx context.Context, e QueryStat) {
	c.add(ctx, func(batch *Batch) { batch.QueryStats = append(batch.QueryStats, e) })
}

// AddEmailEvent enqueues an email lifecycle observation (non-blocking).
//
// Takes e (EmailEvent) which is the email lifecycle observation to enqueue.
//
//nolint:gocritic // hugeParam: a by-value event is the idiomatic public Add API.
func (c *Client) AddEmailEvent(ctx context.Context, e EmailEvent) {
	c.add(ctx, func(batch *Batch) { batch.Emails = append(batch.Emails, e) })
}

// Flush seals and queues the current partial batch (non-blocking).
//
// Returns error which is always nil; the signature matches the flusher interface.
//
// Concurrency: safe for concurrent callers; the seal is serialised by c.mu.
func (c *Client) Flush(ctx context.Context) error {
	c.mu.Lock()
	batch, droppedProfiles := c.sealLocked()
	c.mu.Unlock()
	if batch != nil {
		c.warnDroppedProfiles(ctx, batch, droppedProfiles)
		c.enqueue(ctx, batch)
	}
	return nil
}

// Close stops accepting events, flushes the current batch, drains the queue, and (if it
// owns the connection) closes it.
//
// It waits for the drain up to ctx's deadline.
//
// Returns error which is non-nil only when closing an owned connection fails.
//
// Concurrency: safe for concurrent callers; the first call wins via c.closed.
func (c *Client) Close(ctx context.Context) error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	ctx, l := logger_domain.From(ctx, log)
	if c.ticker != nil {
		c.ticker.Stop()
	}
	close(c.stop)

	c.mu.Lock()
	batch, droppedProfiles := c.sealLocked()
	c.mu.Unlock()
	if batch != nil {
		c.warnDroppedProfiles(ctx, batch, droppedProfiles)
		c.enqueue(ctx, batch)
	}
	c.mu.Lock()
	if !c.sendClosed {
		c.sendClosed = true
		close(c.sendCh)
	}
	c.mu.Unlock()

	if !c.started.Load() {
		return c.closeConnection()
	}

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		l.Warn("telemetry client close timed out before drain completed",
			logger_domain.Error(context.Cause(ctx)))
	}
	if c.cancel != nil {
		c.cancel(errClientClosed)
	}
	grace := time.NewTimer(closeGracePeriod)
	defer grace.Stop()
	select {
	case <-done:
	case <-grace.C:
	}
	return c.closeConnection()
}

// Sent returns the number of events the server acknowledged ingesting.
//
// A batch is counted Sent only once the stream it rode closes with an ack confirming at
// least that many events; events buffered to a stream that later errors before its ack
// are reclassified as Dropped, never Sent.
//
// Returns int64 which is the running count of acknowledged events.
func (c *Client) Sent() int64 { return c.sent.Load() }

// Dropped returns the number of events dropped, whether from a full queue, a send
// failure, a closed client, a stream error before ack, or an ack shortfall.
//
// Returns int64 which is the running count of dropped events.
func (c *Client) Dropped() int64 { return c.dropped.Load() }

// Rejected returns the number of events the server explicitly refused.
//
// A refusal means the stream ended with !ack.OK, or it terminated with a
// codes.Unauthenticated status (bad delivery key). Rejected events are also counted as
// Dropped because they did not land.
//
// Returns int64 which is the running count of rejected events.
func (c *Client) Rejected() int64 { return c.rejected.Load() }

// add applies mutate to the current batch under the lock, sealing and enqueuing the batch
// when it reaches FlushSize events.
//
// It is the shared, non-blocking core behind every public Add method, and is a no-op once
// the client is closed.
//
// Takes mutate (func(*Batch)) which appends the typed event onto the current batch.
//
// Concurrency: safe for concurrent callers; the batch mutation is serialised by c.mu.
func (c *Client) add(ctx context.Context, mutate func(*Batch)) {
	if c.closed.Load() {
		return
	}
	c.mu.Lock()
	if c.currentBatch == nil {
		c.currentBatch = newBatch(&c.config)
	}
	mutate(c.currentBatch)
	c.currentEventCount++
	var ready *Batch
	var droppedProfiles int
	if c.currentEventCount >= c.config.FlushSize {
		ready, droppedProfiles = c.sealLocked()
	}
	c.mu.Unlock()
	if ready != nil {
		c.warnDroppedProfiles(ctx, ready, droppedProfiles)
		c.enqueue(ctx, ready)
	}
}

// sealLocked finalises the current batch and clears it, ready to enqueue.
//
// It stamps the sequence number and sent-at time, then enforces the inline profile budget
// at the seal point so a sealed batch is always within the profile cap. The caller MUST
// hold c.mu. The caller logs the shed-profile count via warnDroppedProfiles AFTER
// releasing c.mu, because a log handler may re-enter an Add method, take c.mu and
// self-deadlock.
//
// Returns *Batch which is the sealed batch, or nil when there is nothing to send.
// Returns int which is the number of profiles whose inline blob was shed.
func (c *Client) sealLocked() (*Batch, int) {
	if c.currentBatch == nil || c.currentEventCount == 0 {
		c.currentBatch = nil
		c.currentEventCount = 0
		return nil, 0
	}
	batch := c.currentBatch
	c.seq++
	batch.Seq = c.seq
	batch.SentAtMs = time.Now().UnixMilli()
	c.currentBatch = nil
	c.currentEventCount = 0
	droppedProfiles := batch.capInlineProfiles()
	return batch, droppedProfiles
}

// warnDroppedProfiles logs, once per sealed batch, that inline profile blobs were shed to
// fit the frame cap.
//
// It MUST be called WITHOUT holding c.mu (see sealLocked).
//
// Takes batch (*Batch) which is the sealed batch the shed profiles belonged to.
// Takes droppedProfiles (int) which is the count of profiles whose blob was shed.
func (*Client) warnDroppedProfiles(ctx context.Context, batch *Batch, droppedProfiles int) {
	if batch == nil || droppedProfiles <= 0 {
		return
	}
	_, l := logger_domain.From(ctx, log)
	l.Warn("inline profile blobs dropped to fit frame cap",
		logger_domain.Int64("seq", batch.Seq), logger_domain.Int("profiles", droppedProfiles))
}

// enqueue offers a sealed batch to the send queue, dropping and counting it when the
// queue is full or already closed. Never blocks.
//
// Takes batch (*Batch) which is the sealed batch to queue for streaming.
//
// Concurrency: safe for concurrent callers; the queue state is guarded by c.mu.
func (c *Client) enqueue(ctx context.Context, batch *Batch) {
	c.mu.Lock()
	if c.sendClosed {
		c.mu.Unlock()
		c.dropped.Add(int64(batch.EventCount()))
		return
	}
	select {
	case c.sendCh <- batch:
		c.mu.Unlock()
	default:
		c.mu.Unlock()
		c.dropped.Add(int64(batch.EventCount()))
		c.warnQueueFullOnce(ctx)
	}
}

// warnQueueFullOnce logs the first time the bounded send queue overflows. Further drops
// are silent (counted in Dropped()) so a sustained overload cannot itself become a log
// storm on the host.
func (c *Client) warnQueueFullOnce(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)
	if c.dropWarned.CompareAndSwap(false, true) {
		l.Warn("telemetry send queue full; dropping batches (further drops counted in Dropped())")
	}
}

// runFlusher periodically flushes partially-filled batches. ctx is the stream context
// derived in Start; it is enriched with the package logger and passed to Flush.
func (c *Client) runFlusher(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	defer func() {
		if r := recover(); r != nil {
			l.Warn("telemetry flusher recovered from panic", logger_domain.Field("panic", r))
		}
	}()
	for {
		select {
		case <-c.stop:
			return
		case <-c.ticker.C:
			_ = c.Flush(ctx)
		}
	}
}

// streamSession is one long-lived client stream plus its unacknowledged event count.
//
// Counting is deferred to reconcile: a batch is not counted Sent the instant SendMsg
// buffers it, because the stream may still error before the terminal IngestAck.
// reconcile, called on every stream close (whether by mid-stream error or normal
// half-close), reads the ack and settles the pending events into sent, dropped or
// rejected.
type streamSession struct {
	// stream is the open client stream, nil before the first send or after a close.
	stream grpc.ClientStream

	// pending is the count of events buffered to stream but not yet acknowledged.
	pending int64
}

// runSender drains the send queue over one long-lived client stream, reopening on error
// behind the circuit breaker.
//
// ctx is the stream context derived in Start; cancelling it via Close tears down any open
// stream. Events buffered to a stream are only counted once that stream closes and its
// ack is reconciled, so the Sent counter never over-counts a batch whose stream later
// fails before acking.
func (c *Client) runSender(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	var session streamSession
	defer func() {
		defer func() {
			if r := recover(); r != nil {
				l.Warn("telemetry sender recovered from panic during teardown",
					logger_domain.String("panic", fmt.Sprint(r)))
			}
		}()
		c.reconcile(ctx, &session)
	}()
	for batch := range c.sendCh {
		c.deliver(ctx, &session, batch)
	}
}

// deliver sends one batch through the circuit breaker, recovering from any panic on the
// third-party gobreaker, gRPC or marshal path.
//
// The recovery means a single bad batch can never crash the host that embeds the
// telemetry client; the goroutine, and thus the queue drain, survives.
//
// Takes session (*streamSession) which is the current stream and its pending tally.
// Takes batch (*Batch) which is the sealed batch to send.
func (c *Client) deliver(ctx context.Context, session *streamSession, batch *Batch) {
	ctx, l := logger_domain.From(ctx, log)
	events := int64(batch.EventCount())
	defer func() {
		if r := recover(); r != nil {
			c.dropped.Add(events)
			l.Warn("telemetry sender recovered from panic", logger_domain.Field("panic", r))
		}
	}()
	_, err := c.breaker.Execute(func() (struct{}, error) {
		return struct{}{}, c.sendOne(ctx, session, batch)
	})
	if err != nil {
		c.dropped.Add(events)
	}
}

// sendOne buffers one batch onto the long-lived stream, opening or reopening it as
// needed, and adds the batch's events to the session's pending tally.
//
// On send failure it reconciles the dead stream (reading whatever ack it can) and clears
// the session so the next call reopens; the failed batch's own events are accounted by
// the caller.
//
// Takes session (*streamSession) which is the current stream and its pending tally.
// Takes batch (*Batch) which is the sealed batch to buffer onto the stream.
//
// Returns error which is non-nil when opening or sending on the stream failed.
func (c *Client) sendOne(ctx context.Context, session *streamSession, batch *Batch) error {
	ctx, l := logger_domain.From(ctx, log)
	data, err := batch.Marshal()
	if err != nil {
		c.dropped.Add(int64(batch.EventCount()))
		l.Warn("telemetry batch dropped before send",
			logger_domain.Int64("seq", batch.Seq), logger_domain.Int(logKeyEvents, batch.EventCount()),
			logger_domain.Error(err))

		return nil
	}
	if session.stream == nil {
		stream, err := c.clientConnection.NewStream(ctx, &serviceDesc.Streams[0], ingestFullMethod,
			grpc.ForceCodec(Codec{}), grpc.MaxCallSendMsgSize(MaxMessageSize))
		if err != nil {
			return fmt.Errorf("telemetry_grpcfb: open stream: %w", err)
		}
		session.stream = stream
		session.pending = 0
	}
	if err := session.stream.SendMsg(preEncodedBatch{data: data}); err != nil {
		c.reconcile(ctx, session)
		return fmt.Errorf("telemetry_grpcfb: send batch: %w", err)
	}
	session.pending += int64(batch.EventCount())
	return nil
}

// reconcile closes the session's stream, reads its terminal IngestAck and settles the
// pending events.
//
// On a clean ack (ok and events >= pending) the pending events count Sent; a
// server-confirmed shortfall (events < pending) reclassifies the missing events as
// Dropped; a server rejection (!ok, or an Unauthenticated stream status) counts the
// pending events as both Rejected and Dropped and logs a warning. It is safe to call with
// a nil stream (a no-op) and resets the session afterwards.
//
// Takes session (*streamSession) which is the stream to close and its pending tally.
func (c *Client) reconcile(ctx context.Context, session *streamSession) {
	_, l := logger_domain.From(ctx, log)
	if session.stream == nil {
		session.pending = 0
		return
	}
	pending := session.pending
	closeErr := session.stream.CloseSend()
	var ack IngestAck
	recvErr := session.stream.RecvMsg(&ack)
	session.stream = nil
	session.pending = 0
	if pending == 0 {
		return
	}

	if status.Code(recvErr) == codes.Unauthenticated {
		c.rejected.Add(pending)
		c.dropped.Add(pending)
		l.Warn("telemetry stream rejected",
			logger_domain.String("reason", "unauthenticated"), logger_domain.Int64(logKeyEvents, pending),
			logger_domain.Error(recvErr))
		return
	}
	if recvErr != nil && !errors.Is(recvErr, io.EOF) {
		c.dropped.Add(pending)
		l.Warn("telemetry stream closed before ack",
			logger_domain.Int64(logKeyEvents, pending),
			logger_domain.Error(errors.Join(recvErr, closeErr)))
		return
	}
	if !ack.OK {
		c.rejected.Add(pending)
		c.dropped.Add(pending)
		l.Warn("telemetry stream rejected",
			logger_domain.String("reason", "ack_not_ok"), logger_domain.Int64(logKeyEvents, pending),
			logger_domain.String("message", ack.Message))
		return
	}
	if ack.Events < pending {
		shortfall := pending - ack.Events
		c.sent.Add(ack.Events)
		c.dropped.Add(shortfall)
		l.Warn("telemetry ack short of streamed events",
			logger_domain.Int64("streamed", pending), logger_domain.Int64("acked", ack.Events),
			logger_domain.Int64("dropped", shortfall))
		return
	}
	c.sent.Add(pending)
}

// closeConnection closes the underlying connection when the Client owns it.
//
// Returns error which is non-nil only when closing an owned connection failed.
func (c *Client) closeConnection() error {
	if c.ownsConnection && c.clientConnection != nil {
		if err := c.clientConnection.Close(); err != nil {
			return fmt.Errorf("telemetry_grpcfb: close connection: %w", err)
		}
	}
	return nil
}

// New builds a Client that streams over an existing connection.
//
// The caller owns clientConnection and must supply a non-nil connection. Call Start
// before producing events and Close to drain and stop.
//
// Takes clientConnection (*grpc.ClientConn) which is the caller-owned connection to
// stream over.
// Takes config (Config) which configures the client; zero fields take their defaults.
//
// Returns *Client which streams over clientConnection but does not close it.
func New(clientConnection *grpc.ClientConn, config Config) *Client {
	return newClient(clientConnection, false, config)
}

// Dial opens its own connection to target, which the Client owns and closes on Close.
//
// dialOpts are appended to the telemetry defaults (insecure-free, so the caller supplies
// transport credentials). It returns ErrEmptyTarget when target is empty.
//
// Takes target (string) which is the dial address of the telemetry sink.
// Takes config (Config) which configures the client; zero fields take their defaults.
// Takes dialOpts (...grpc.DialOption) which are appended to the telemetry dial defaults.
//
// Returns *Client which owns and closes the dialled connection.
// Returns error which is ErrEmptyTarget for an empty target, or a dial failure.
func Dial(target string, config Config, dialOpts ...grpc.DialOption) (*Client, error) {
	if target == "" {
		return nil, ErrEmptyTarget
	}
	opts := append([]grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.ForceCodec(Codec{})),
	}, dialOpts...)
	clientConnection, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, fmt.Errorf("telemetry_grpcfb: dial %q: %w", target, err)
	}
	return newClient(clientConnection, true, config), nil
}

// newClient builds a Client over clientConnection with defaults applied and the send-path
// circuit breaker configured.
//
// It is the shared constructor behind New and Dial, differing only in whether the Client
// owns the connection.
//
// Takes clientConnection (*grpc.ClientConn) which is the connection to stream over.
// Takes ownsConnection (bool) which is true when Close must close clientConnection.
// Takes config (Config) which configures the client; zero fields take their defaults.
//
// Returns *Client which is ready to Start.
func newClient(clientConnection *grpc.ClientConn, ownsConnection bool, config Config) *Client {
	config = config.withDefaults()
	breakerConfig := config.Breaker
	maxFailures := defaultMaxFailures
	openTimeout := defaultOpenTimeout
	if breakerConfig != nil {
		if breakerConfig.MaxConsecutiveFailures > 0 {
			maxFailures = breakerConfig.MaxConsecutiveFailures
		}
		if breakerConfig.OpenTimeout > 0 {
			openTimeout = breakerConfig.OpenTimeout
		}
	}
	return &Client{
		config:           config,
		clientConnection: clientConnection,
		ownsConnection:   ownsConnection,
		stop:             make(chan struct{}),
		sendCh:           make(chan *Batch, config.MaxQueuedBatches),
		breaker: gobreaker.NewCircuitBreaker[struct{}](gobreaker.Settings{
			Name:        "telemetry-grpcfb-send",
			MaxRequests: 1,
			Timeout:     openTimeout,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return int(counts.ConsecutiveFailures) >= maxFailures
			},
			IsExcluded: func(err error) bool {
				return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
			},
		}),
	}
}

// newBatch starts a frame carrying the site, key, source and the emitter's identity.
//
// Takes config (*Config) which holds the values stamped on every frame.
//
// Returns *Batch which is an empty frame ready for events.
func newBatch(config *Config) *Batch {
	identity := config.Identity
	return &Batch{
		SiteID: config.SiteID, APIKey: config.APIKey, Source: config.Source,
		InstanceID: identity.InstanceID, Hostname: identity.Hostname,
		ServiceName: identity.ServiceName, ServiceVersion: identity.ServiceVersion,
		Environment: identity.Environment, Region: identity.Region,
		StartedAtMs: identity.StartedAtMs, PID: identity.PID,
	}
}
