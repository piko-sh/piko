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

package analytics_collector_ga4

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/retry"
	"piko.sh/piko/wdk/analytics"
	"piko.sh/piko/wdk/maths"
)

const (
	// defaultBatchSize is the number of events buffered before
	// triggering a flush signal.
	defaultBatchSize = 25

	// maxEventsPerRequest is the hard limit imposed by the GA4
	// Measurement Protocol.
	maxEventsPerRequest = 25

	// defaultFlushInterval is the time between automatic flushes.
	defaultFlushInterval = 5 * time.Second

	// defaultTimeout is the HTTP client timeout.
	defaultTimeout = 10 * time.Second

	// collectorName identifies this collector in logs and metrics.
	collectorName = "ga4"

	// paramPageLocation is the GA4 parameter key for the full
	// page URL.
	paramPageLocation = "page_location"

	// paramsInitialCapacity is the pre-allocated size for GA4
	// event parameter maps, covering the typical number of standard
	// fields (page_location, page_referrer, language, duration_ms,
	// status_code, action_name, currency, value, timestamp_micros,
	// plus headroom for user properties).
	paramsInitialCapacity = 12

	// productionEndpoint is the GA4 Measurement Protocol endpoint.
	productionEndpoint = "https://www.google-analytics.com/mp/collect"

	// debugEndpoint validates events without recording them.
	debugEndpoint = "https://www.google-analytics.com/debug/mp/collect"

	// sha256DigestSize is the length of a SHA-256 digest in bytes.
	sha256DigestSize = sha256.Size

	// httpStatusErrorThreshold is the lowest HTTP status code
	// treated as an error.
	httpStatusErrorThreshold = 400
)

// eventSnapshot is a pre-computed copy of an analytics event in
// GA4-ready form. Created in Collect to avoid retaining the pooled
// Event pointer.
type eventSnapshot struct {
	// timestamp is when the event occurred.
	timestamp time.Time

	// params holds the GA4 event parameters.
	params map[string]any

	// name is the GA4 event name (e.g. "page_view", "purchase").
	name string

	// clientID identifies the client for the Measurement Protocol.
	clientID string

	// userID is the authenticated user ID, empty if anonymous.
	userID string
}

// payload is the top-level JSON body sent to the GA4 Measurement
// Protocol endpoint.
type payload struct {
	// ClientID is the client identifier (required by GA4).
	ClientID string `json:"client_id"`

	// UserID is the authenticated user identifier.
	UserID string `json:"user_id,omitempty"`

	// Events is the array of events to send.
	Events []ga4Event `json:"events"`

	// TimestampMicros is the event timestamp in microseconds.
	TimestampMicros int64 `json:"timestamp_micros,omitempty"`
}

// ga4Event is a single event in the GA4 events array.
type ga4Event struct {
	// Params holds the event parameters.
	Params map[string]any `json:"params,omitempty"`

	// Name is the GA4 event name.
	Name string `json:"name"`
}

// Option configures a [Collector].
type Option func(*Collector)

// WithBatchSize sets the maximum number of events buffered before
// triggering a flush. The value is clamped to the GA4 maximum of 25
// events per request.
//
// Takes size (int) which is the batch capacity.
//
// Returns Option which configures the batch size.
func WithBatchSize(size int) Option {
	return func(c *Collector) {
		if size > 0 {
			c.batchSize = min(size, maxEventsPerRequest)
		}
	}
}

// WithFlushInterval sets the time between automatic batch
// flushes. Defaults to 5 seconds.
//
// Takes d (time.Duration) which is the flush interval.
//
// Returns Option which configures the interval.
func WithFlushInterval(d time.Duration) Option {
	return func(c *Collector) {
		if d > 0 {
			c.flushInterval = d
		}
	}
}

// WithTimeout sets the HTTP client timeout for GA4 POSTs.
// Defaults to 10 seconds.
//
// Takes d (time.Duration) which is the client timeout.
//
// Returns Option which configures the timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Collector) {
		if d > 0 {
			c.client.Timeout = d
		}
	}
}

// WithClientIDFunc sets a custom function to derive the GA4
// client_id from request data. The default function hashes the
// client IP and user agent with SHA-256.
//
// Takes fn (func(clientIP, userAgent string) string) which derives
// the client ID.
//
// Returns Option which configures the client ID function.
func WithClientIDFunc(fn func(clientIP, userAgent string) string) Option {
	return func(c *Collector) {
		if fn != nil {
			c.clientIDFunc = fn
		}
	}
}

// WithRetry enables retry with exponential backoff for failed
// batch sends. Only retryable errors (network failures, 5xx) are
// retried; permanent errors fail immediately.
//
// Takes config (analytics.RetryConfig) which configures the retry
// behaviour.
//
// Returns Option which configures the retry.
func WithRetry(config analytics.RetryConfig) Option {
	return func(c *Collector) {
		c.retryConfig = &config
	}
}

// WithCircuitBreaker enables a circuit breaker that stops
// sending batches after consecutive failures. The circuit reopens
// after the timeout expires and a probe request succeeds.
//
// Takes config (analytics.CircuitBreakerConfig) which configures
// the circuit breaker.
//
// Returns Option which configures the circuit breaker.
func WithCircuitBreaker(config analytics.CircuitBreakerConfig) Option {
	return func(c *Collector) {
		c.circuitBreakerConfig = &config
	}
}

// WithDebug enables the GA4 debug endpoint which validates
// events but does not record them in Google Analytics.
//
// Takes debug (bool) which enables or disables debug mode.
//
// Returns Option which configures debug mode.
func WithDebug(debug bool) Option {
	return func(c *Collector) {
		c.debug = debug
	}
}

// Collector sends analytics events to the Google Analytics 4
// Measurement Protocol. Events are buffered internally and flushed
// when the batch reaches batchSize or the flushInterval expires.
type Collector struct {
	// batcher manages the buffer, flush loop, and lifecycle.
	batcher *analytics_domain.Batcher[eventSnapshot]

	// paramsPool recycles map[string]any instances to avoid
	// per-event allocation in Collect.
	paramsPool sync.Pool

	// client is the HTTP client used for batch POST requests.
	client *http.Client

	// clientIDFunc derives a GA4 client_id from request data.
	clientIDFunc func(clientIP, userAgent string) string

	// retryConfig holds optional retry settings. Nil disables retry.
	retryConfig *retry.Config

	// circuitBreakerConfig holds optional circuit breaker settings.
	// Nil disables the circuit breaker.
	circuitBreakerConfig *analytics_domain.CircuitBreakerConfig

	// endpoint is the full GA4 Measurement Protocol URL including
	// measurement_id and api_secret query parameters.
	endpoint string

	// jsonBuffer is a reusable buffer for JSON encoding.
	jsonBuffer bytes.Buffer

	// flushInterval is the time between automatic timer-based flushes.
	flushInterval time.Duration

	// batchSize is the number of events that triggers an immediate
	// flush signal.
	batchSize int

	// debug enables the GA4 validation endpoint.
	debug bool
}

// NewCollector creates an analytics collector that sends events to
// the GA4 Measurement Protocol endpoint.
//
// Takes measurementID (string) which is the GA4 property measurement
// ID (e.g. "G-XXXXXXXXXX").
// Takes apiSecret (string) which is the Measurement Protocol API
// secret created in GA4 Admin.
// Takes opts (...Option) which configure the collector.
//
// Returns analytics.Collector which sends events to GA4.
func NewCollector(measurementID, apiSecret string, opts ...Option) analytics.Collector {
	if measurementID == "" {
		panic("analytics: GA4 measurementID must not be empty")
	}
	if apiSecret == "" {
		panic("analytics: GA4 apiSecret must not be empty")
	}

	c := &Collector{
		client:        &http.Client{Timeout: defaultTimeout},
		clientIDFunc:  defaultClientID,
		batchSize:     defaultBatchSize,
		flushInterval: defaultFlushInterval,
	}
	for _, opt := range opts {
		opt(c)
	}

	baseURL := productionEndpoint
	if c.debug {
		baseURL = debugEndpoint
	}
	params := url.Values{}
	params.Set("measurement_id", measurementID)
	params.Set("api_secret", apiSecret)
	c.endpoint = baseURL + "?" + params.Encode()

	c.batcher = analytics_domain.NewBatcher[eventSnapshot](
		analytics_domain.BatcherConfig{
			Name:           collectorName,
			BatchSize:      c.batchSize,
			FlushInterval:  c.flushInterval,
			Retry:          c.retryConfig,
			CircuitBreaker: c.circuitBreakerConfig,
		},
		c.sendBatch,
	)
	return c
}

// Start launches the background flush loop. Called by the analytics
// Service after registration.
func (c *Collector) Start(ctx context.Context) {
	c.batcher.Start(ctx)
}

// Collect copies the event data into the internal buffer. When the
// buffer reaches batchSize, the flush goroutine is signalled to send
// the batch asynchronously. Collect itself never performs I/O.
//
// Takes event (*analytics_dto.Event) which carries the event data.
//
// Returns error which is always nil.
func (c *Collector) Collect(_ context.Context, event *analytics_dto.Event) error {
	snap := eventSnapshot{
		clientID:  c.clientIDFunc(event.ClientIP, event.UserAgent),
		userID:    event.UserID,
		timestamp: event.Timestamp,
		params:    c.acquireParams(),
	}

	snap.name = resolveEventName(event)
	mapParams(snap.params, event)

	snap.params["timestamp_micros"] = event.Timestamp.UnixMicro()

	for key, value := range event.Properties {
		snap.params[key] = value
	}

	c.batcher.Add(snap)
	return nil
}

// resolveEventName determines the GA4 event name from the
// analytics event type and fields.
//
// Takes event (*analytics_dto.Event) which carries the event data.
//
// Returns string which is the GA4 event name.
func resolveEventName(event *analytics_dto.Event) string {
	switch {
	case event.EventName != "":
		return event.EventName
	case event.Type == analytics_dto.EventPageView:
		return "page_view"
	case event.Type == analytics_dto.EventAction:
		if event.ActionName != "" {
			return event.ActionName
		}
		return "server_action"
	default:
		return "custom_event"
	}
}

// mapParams populates the GA4 parameter map from standard event
// fields and revenue data.
//
// Takes params (map[string]any) which is the target parameter map.
// Takes event (*analytics_dto.Event) which provides the source data.
func mapParams(params map[string]any, event *analytics_dto.Event) {
	switch {
	case event.URL != "" && strings.HasPrefix(event.URL, "http"):
		params[paramPageLocation] = event.URL
	case event.Hostname != "" && event.URL != "":
		params[paramPageLocation] = "https://" + event.Hostname + event.URL
	case event.Hostname != "" && event.Path != "":
		params[paramPageLocation] = "https://" + event.Hostname + event.Path
	case event.URL != "":
		params[paramPageLocation] = event.URL
	}
	if event.Referrer != "" {
		params["page_referrer"] = event.Referrer
	}
	if event.Locale != "" {
		params["language"] = event.Locale
	}
	if event.Duration > 0 {
		params["duration_ms"] = float64(event.Duration) / float64(time.Millisecond)
	}
	if event.StatusCode != 0 {
		params["status_code"] = event.StatusCode
	}
	if event.ActionName != "" {
		params["action_name"] = event.ActionName
	}

	if event.Revenue != nil {
		mapRevenue(params, event.Revenue)
	}
}

// mapRevenue extracts currency and monetary value from a Money
// pointer into GA4 parameters.
//
// Takes params (map[string]any) which is the target parameter map.
// Takes revenue (*maths.Money) which provides the monetary data.
func mapRevenue(params map[string]any, revenue *maths.Money) {
	revenueValue := *revenue
	if currencyCode, currencyError := revenueValue.CurrencyCode(); currencyError == nil {
		params["currency"] = currencyCode
	}
	if amount, amountError := revenueValue.Amount(); amountError == nil {
		if floatValue, floatError := amount.Float64(); floatError == nil {
			params["value"] = floatValue
		}
	}
}

// Flush sends any buffered events to the GA4 endpoint.
//
// Returns error when the POST fails.
func (c *Collector) Flush(ctx context.Context) error {
	return c.batcher.Flush(ctx)
}

// Close stops the flush timer and releases resources. Any remaining
// buffered events should be flushed via Flush before calling Close.
// Safe to call multiple times.
//
// Returns error which is always nil.
func (c *Collector) Close(_ context.Context) error {
	return c.batcher.Close()
}

// Name returns the collector name.
//
// Returns string which identifies this collector.
func (*Collector) Name() string {
	return collectorName
}

// sendBatch groups events by clientID and POSTs each group to the
// GA4 Measurement Protocol endpoint. GA4 requires a single client_id
// per request body, so different clients produce separate requests.
//
// Takes batch ([]eventSnapshot) which holds the events to send.
//
// Returns error which aggregates failures from all chunk sends.
func (c *Collector) sendBatch(ctx context.Context, batch []eventSnapshot) error {
	ctx, _ = logger_domain.From(ctx, log)
	defer c.releaseSnapshotParams(batch)

	type groupKey struct{ clientID, userID string }
	groups := make(map[groupKey][]eventSnapshot)
	for index := range batch {
		key := groupKey{clientID: batch[index].clientID, userID: batch[index].userID}
		groups[key] = append(groups[key], batch[index])
	}

	var errs []error
	for key, snapshots := range groups {
		if ctx.Err() != nil {
			errs = append(errs, ctx.Err())
			break
		}

		for chunkStart := 0; chunkStart < len(snapshots); chunkStart += maxEventsPerRequest {
			if ctx.Err() != nil {
				errs = append(errs, ctx.Err())
				break
			}

			chunkEnd := min(chunkStart+maxEventsPerRequest, len(snapshots))
			chunk := snapshots[chunkStart:chunkEnd]

			if err := c.sendChunk(ctx, key.clientID, key.userID, chunk); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// sendChunk POSTs a single GA4 request body for the given clientID,
// userID, and event chunk.
//
// Takes clientID (string) which identifies the GA4 client.
// Takes userID (string) which is the authenticated user ID.
// Takes chunk ([]eventSnapshot) which holds the events to send.
//
// Returns error when encoding or the HTTP request fails.
func (c *Collector) sendChunk(ctx context.Context, clientID, userID string, chunk []eventSnapshot) (returnErr error) {
	defer func() {
		if r := recover(); r != nil {
			returnErr = goroutine.HandlePanicRecovery(ctx, "analytics_collector_ga4.sendChunk", r)
		}
	}()

	ctx, l := logger_domain.From(ctx, log)
	batchSize.Record(ctx, int64(len(chunk)))

	if err := c.encodeChunk(clientID, userID, chunk); err != nil {
		errorCount.Add(ctx, 1)
		l.Warn("Analytics GA4 JSON encoding failed", logger_domain.Error(err))
		return fmt.Errorf("encoding analytics GA4 batch: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(c.jsonBuffer.Bytes()))
	if err != nil {
		errorCount.Add(ctx, 1)
		return fmt.Errorf("creating analytics GA4 request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	start := time.Now()
	response, err := c.client.Do(request)
	duration := float64(time.Since(start)) / float64(time.Millisecond)

	if err != nil {
		errorCount.Add(ctx, 1)
		l.Warn("Analytics GA4 POST failed", logger_domain.Error(err))
		return fmt.Errorf("posting analytics GA4 batch: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}()

	if response.StatusCode >= httpStatusErrorThreshold {
		errorCount.Add(ctx, 1)
		l.Warn("Analytics GA4 returned error status",
			logger_domain.Int("status_code", response.StatusCode))
		return fmt.Errorf("analytics GA4 returned status %d", response.StatusCode)
	}

	sendCount.Add(ctx, 1)
	sendDuration.Record(ctx, duration)

	return nil
}

// encodeChunk builds the GA4 payload from the event chunk and
// encodes it into the reusable JSON buffer.
//
// Takes clientID (string) which identifies the GA4 client.
// Takes userID (string) which is the authenticated user ID.
// Takes chunk ([]eventSnapshot) which holds the events to encode.
//
// Returns error when JSON encoding fails.
func (c *Collector) encodeChunk(clientID, userID string, chunk []eventSnapshot) error {
	events := make([]ga4Event, len(chunk))
	for index, snap := range chunk {
		events[index] = ga4Event{
			Name:   snap.name,
			Params: snap.params,
		}
	}

	p := payload{
		ClientID:        clientID,
		UserID:          userID,
		TimestampMicros: chunk[0].timestamp.UnixMicro(),
		Events:          events,
	}

	c.jsonBuffer.Reset()
	return json.NewEncoder(&c.jsonBuffer).Encode(p)
}

// releaseSnapshotParams returns all pooled params maps from a batch
// back to the pool.
//
// Takes batch ([]eventSnapshot) which holds the snapshots to release.
func (c *Collector) releaseSnapshotParams(batch []eventSnapshot) {
	for index := range batch {
		if batch[index].params != nil {
			c.releaseParams(batch[index].params)
			batch[index].params = nil
		}
	}
}

// acquireParams returns a cleared map from the pool, or allocates a
// new one on pool miss.
//
// Returns map[string]any which is the reusable parameter map.
func (c *Collector) acquireParams() map[string]any {
	if pooled, ok := c.paramsPool.Get().(map[string]any); ok {
		clear(pooled)
		return pooled
	}
	return make(map[string]any, paramsInitialCapacity)
}

// releaseParams returns a params map to the pool.
//
// Takes params (map[string]any) which is the map to return.
func (c *Collector) releaseParams(params map[string]any) {
	c.paramsPool.Put(params)
}

// sha256Pool reuses SHA-256 hashers to avoid a 96-byte allocation
// on every Collect call.
var sha256Pool = sync.Pool{
	New: func() any { return sha256.New() },
}

// separatorByte is the delimiter written between client IP and user
// agent when computing the client ID hash.
var separatorByte = []byte("|")

// defaultClientID produces a deterministic pseudo-anonymous
// client identifier by hashing the client IP and user agent.
// The hasher is pooled to avoid per-call allocation.
//
// Takes clientIP (string) which is the client's IP address.
// Takes userAgent (string) which is the client's user agent.
//
// Returns string which is the hex-encoded SHA-256 hash.
func defaultClientID(clientIP, userAgent string) string {
	if clientIP == "" && userAgent == "" {
		return "anonymous"
	}

	hasher, ok := sha256Pool.Get().(hash.Hash)
	if !ok {
		hasher = sha256.New()
	}

	hasher.Reset()
	_, _ = io.WriteString(hasher, clientIP)
	_, _ = hasher.Write(separatorByte)
	_, _ = io.WriteString(hasher, userAgent)

	var digest [sha256DigestSize]byte
	hasher.Sum(digest[:0])
	sha256Pool.Put(hasher)

	return hex.EncodeToString(digest[:])
}
