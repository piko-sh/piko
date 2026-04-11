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

package analytics_adapters

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
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/retry"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/maths"
)

const (
	// defaultGA4BatchSize is the number of events buffered before
	// triggering a flush signal.
	defaultGA4BatchSize = 25

	// maxGA4EventsPerRequest is the hard limit imposed by the GA4
	// Measurement Protocol.
	maxGA4EventsPerRequest = 25

	// defaultGA4FlushInterval is the time between automatic flushes.
	defaultGA4FlushInterval = 5 * time.Second

	// defaultGA4Timeout is the HTTP client timeout.
	defaultGA4Timeout = 10 * time.Second

	// ga4CollectorName identifies this collector in logs and metrics.
	ga4CollectorName = "ga4"

	// ga4ParamPageLocation is the GA4 parameter key for the full
	// page URL.
	ga4ParamPageLocation = "page_location"

	// ga4ParamsInitialCapacity is the pre-allocated size for GA4
	// event parameter maps, covering the typical number of standard
	// fields (page_location, page_referrer, language, duration_ms,
	// status_code, action_name, currency, value, timestamp_micros,
	// plus headroom for user properties).
	ga4ParamsInitialCapacity = 12

	// ga4ProductionEndpoint is the GA4 Measurement Protocol endpoint.
	ga4ProductionEndpoint = "https://www.google-analytics.com/mp/collect"

	// ga4DebugEndpoint validates events without recording them.
	ga4DebugEndpoint = "https://www.google-analytics.com/debug/mp/collect"

	// sha256DigestSize is the length of a SHA-256 digest in bytes.
	sha256DigestSize = sha256.Size
)

// ga4EventSnapshot is a pre-computed copy of an analytics event in
// GA4-ready form. Created in Collect to avoid retaining the pooled
// Event pointer.
type ga4EventSnapshot struct {
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

// ga4Payload is the top-level JSON body sent to the GA4 Measurement
// Protocol endpoint.
type ga4Payload struct {
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

// GA4Option configures a GA4Collector.
type GA4Option func(*GA4Collector)

// WithGA4BatchSize sets the maximum number of events buffered before
// triggering a flush. The value is clamped to the GA4 maximum of 25
// events per request.
//
// Takes size (int) which is the batch capacity.
//
// Returns GA4Option which configures the batch size.
func WithGA4BatchSize(size int) GA4Option {
	return func(gc *GA4Collector) {
		if size > 0 {
			gc.batchSize = min(size, maxGA4EventsPerRequest)
		}
	}
}

// WithGA4FlushInterval sets the time between automatic batch
// flushes. Defaults to 5 seconds.
//
// Takes d (time.Duration) which is the flush interval.
//
// Returns GA4Option which configures the interval.
func WithGA4FlushInterval(d time.Duration) GA4Option {
	return func(gc *GA4Collector) {
		if d > 0 {
			gc.flushInterval = d
		}
	}
}

// WithGA4Timeout sets the HTTP client timeout for GA4 POSTs.
// Defaults to 10 seconds.
//
// Takes d (time.Duration) which is the client timeout.
//
// Returns GA4Option which configures the timeout.
func WithGA4Timeout(d time.Duration) GA4Option {
	return func(gc *GA4Collector) {
		if d > 0 {
			gc.client.Timeout = d
		}
	}
}

// WithGA4ClientIDFunc sets a custom function to derive the GA4
// client_id from request data. The default function hashes the
// client IP and user agent with SHA-256.
//
// Takes fn (func(clientIP, userAgent string) string) which derives
// the client ID.
//
// Returns GA4Option which configures the client ID function.
func WithGA4ClientIDFunc(fn func(clientIP, userAgent string) string) GA4Option {
	return func(gc *GA4Collector) {
		if fn != nil {
			gc.clientIDFunc = fn
		}
	}
}

// WithGA4Retry enables retry with exponential backoff for failed
// batch sends. Only retryable errors (network failures, 5xx) are
// retried; permanent errors fail immediately.
//
// Takes config (retry.Config) which configures the retry behaviour.
//
// Returns GA4Option which configures the retry.
func WithGA4Retry(config retry.Config) GA4Option {
	return func(gc *GA4Collector) {
		gc.retryConfig = &config
	}
}

// WithGA4CircuitBreaker enables a circuit breaker that stops
// sending batches after consecutive failures. The circuit reopens
// after the timeout expires and a probe request succeeds.
//
// Takes config (analytics_domain.CircuitBreakerConfig) which
// configures the circuit breaker.
//
// Returns GA4Option which configures the circuit breaker.
func WithGA4CircuitBreaker(config analytics_domain.CircuitBreakerConfig) GA4Option {
	return func(gc *GA4Collector) {
		gc.circuitBreakerConfig = &config
	}
}

// WithGA4Debug enables the GA4 debug endpoint which validates
// events but does not record them in Google Analytics.
//
// Takes debug (bool) which enables or disables debug mode.
//
// Returns GA4Option which configures debug mode.
func WithGA4Debug(debug bool) GA4Option {
	return func(gc *GA4Collector) {
		gc.debug = debug
	}
}

// GA4Collector sends analytics events to the Google Analytics 4
// Measurement Protocol. Events are buffered internally and flushed
// when the batch reaches batchSize or the flushInterval expires.
type GA4Collector struct {
	// batcher manages the buffer, flush loop, and lifecycle.
	batcher *analytics_domain.Batcher[ga4EventSnapshot]

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

// NewGA4Collector creates a collector that sends events to the GA4
// Measurement Protocol endpoint.
//
// Takes measurementID (string) which is the GA4 property measurement
// ID (e.g. "G-XXXXXXXXXX").
// Takes apiSecret (string) which is the Measurement Protocol API
// secret created in GA4 Admin.
// Takes opts (...GA4Option) which configure the collector.
//
// Returns *GA4Collector which is the configured collector.
func NewGA4Collector(measurementID, apiSecret string, opts ...GA4Option) *GA4Collector {
	if measurementID == "" {
		panic("analytics: GA4 measurementID must not be empty")
	}
	if apiSecret == "" {
		panic("analytics: GA4 apiSecret must not be empty")
	}

	gc := &GA4Collector{
		client:        &http.Client{Timeout: defaultGA4Timeout},
		clientIDFunc:  defaultGA4ClientID,
		batchSize:     defaultGA4BatchSize,
		flushInterval: defaultGA4FlushInterval,
	}
	for _, opt := range opts {
		opt(gc)
	}

	baseURL := ga4ProductionEndpoint
	if gc.debug {
		baseURL = ga4DebugEndpoint
	}
	params := url.Values{}
	params.Set("measurement_id", measurementID)
	params.Set("api_secret", apiSecret)
	gc.endpoint = baseURL + "?" + params.Encode()

	gc.batcher = analytics_domain.NewBatcher[ga4EventSnapshot](
		analytics_domain.BatcherConfig{
			Name:           ga4CollectorName,
			BatchSize:      gc.batchSize,
			FlushInterval:  gc.flushInterval,
			Retry:          gc.retryConfig,
			CircuitBreaker: gc.circuitBreakerConfig,
		},
		gc.sendBatch,
	)
	return gc
}

// Start launches the background flush loop. Called by the analytics
// Service after registration.
func (gc *GA4Collector) Start(ctx context.Context) {
	gc.batcher.Start(ctx)
}

// Collect copies the event data into the internal buffer. When the
// buffer reaches batchSize, the flush goroutine is signalled to send
// the batch asynchronously. Collect itself never performs I/O.
//
// Takes event (*analytics_dto.Event) which carries the event data.
//
// Returns error which is always nil.
func (gc *GA4Collector) Collect(_ context.Context, event *analytics_dto.Event) error {
	snap := ga4EventSnapshot{
		clientID:  gc.clientIDFunc(event.ClientIP, event.UserAgent),
		userID:    event.UserID,
		timestamp: event.Timestamp,
		params:    gc.acquireParams(),
	}

	snap.name = resolveGA4EventName(event)
	mapGA4Params(snap.params, event)

	snap.params["timestamp_micros"] = event.Timestamp.UnixMicro()

	for key, value := range event.Properties {
		snap.params[key] = value
	}

	gc.batcher.Add(snap)
	return nil
}

// resolveGA4EventName determines the GA4 event name from the
// analytics event type and fields.
//
// Takes event (*analytics_dto.Event) which carries the event data.
//
// Returns string which is the GA4 event name.
func resolveGA4EventName(event *analytics_dto.Event) string {
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

// mapGA4Params populates the GA4 parameter map from standard event
// fields and revenue data.
//
// Takes params (map[string]any) which is the target parameter map.
// Takes event (*analytics_dto.Event) which provides the source data.
func mapGA4Params(params map[string]any, event *analytics_dto.Event) {
	switch {
	case event.URL != "" && strings.HasPrefix(event.URL, "http"):
		params[ga4ParamPageLocation] = event.URL
	case event.Hostname != "" && event.URL != "":
		params[ga4ParamPageLocation] = "https://" + event.Hostname + event.URL
	case event.Hostname != "" && event.Path != "":
		params[ga4ParamPageLocation] = "https://" + event.Hostname + event.Path
	case event.URL != "":
		params[ga4ParamPageLocation] = event.URL
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
		mapGA4Revenue(params, event.Revenue)
	}
}

// mapGA4Revenue extracts currency and monetary value from a Money
// pointer into GA4 parameters.
//
// Takes params (map[string]any) which is the target parameter map.
// Takes revenue (*maths.Money) which provides the monetary data.
func mapGA4Revenue(params map[string]any, revenue *maths.Money) {
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
func (gc *GA4Collector) Flush(ctx context.Context) error {
	return gc.batcher.Flush(ctx)
}

// Close stops the flush timer and releases resources. Any remaining
// buffered events should be flushed via Flush before calling Close.
// Safe to call multiple times.
//
// Returns error which is always nil.
func (gc *GA4Collector) Close(_ context.Context) error {
	return gc.batcher.Close()
}

// Name returns the collector name.
//
// Returns string which identifies this collector.
func (*GA4Collector) Name() string {
	return ga4CollectorName
}

// sendBatch groups events by clientID and POSTs each group to the
// GA4 Measurement Protocol endpoint. GA4 requires a single client_id
// per request body, so different clients produce separate requests.
//
// Takes batch ([]ga4EventSnapshot) which holds the events to send.
//
// Returns error which aggregates failures from all chunk sends.
func (gc *GA4Collector) sendBatch(ctx context.Context, batch []ga4EventSnapshot) error {
	defer gc.releaseSnapshotParams(batch)

	type groupKey struct{ clientID, userID string }
	groups := make(map[groupKey][]ga4EventSnapshot)
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

		for chunkStart := 0; chunkStart < len(snapshots); chunkStart += maxGA4EventsPerRequest {
			if ctx.Err() != nil {
				errs = append(errs, ctx.Err())
				break
			}

			chunkEnd := min(chunkStart+maxGA4EventsPerRequest, len(snapshots))
			chunk := snapshots[chunkStart:chunkEnd]

			if err := gc.sendChunk(ctx, key.clientID, key.userID, chunk); err != nil {
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
// Takes chunk ([]ga4EventSnapshot) which holds the events to send.
//
// Returns error when encoding or the HTTP request fails.
func (gc *GA4Collector) sendChunk(ctx context.Context, clientID, userID string, chunk []ga4EventSnapshot) error {
	ctx, l := logger_domain.From(ctx, log)
	ga4BatchSize.Record(ctx, int64(len(chunk)))

	events := make([]ga4Event, len(chunk))
	for index, snap := range chunk {
		events[index] = ga4Event{
			Name:   snap.name,
			Params: snap.params,
		}
	}

	payload := ga4Payload{
		ClientID:        clientID,
		UserID:          userID,
		TimestampMicros: chunk[0].timestamp.UnixMicro(),
		Events:          events,
	}

	gc.jsonBuffer.Reset()
	encoder := json.NewEncoder(&gc.jsonBuffer)
	if err := encoder.Encode(payload); err != nil {
		ga4ErrorCount.Add(ctx, 1)
		l.Warn("Analytics GA4 JSON encoding failed", logger_domain.Error(err))
		return fmt.Errorf("encoding analytics GA4 batch: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, gc.endpoint, bytes.NewReader(gc.jsonBuffer.Bytes()))
	if err != nil {
		ga4ErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating analytics GA4 request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	start := time.Now()
	response, err := gc.client.Do(request)
	duration := float64(time.Since(start)) / float64(time.Millisecond)

	if err != nil {
		ga4ErrorCount.Add(ctx, 1)
		l.Warn("Analytics GA4 POST failed", logger_domain.Error(err))
		return fmt.Errorf("posting analytics GA4 batch: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}()

	if response.StatusCode >= httpStatusErrorThreshold {
		ga4ErrorCount.Add(ctx, 1)
		l.Warn("Analytics GA4 returned error status",
			logger_domain.Int("status_code", response.StatusCode))
		return fmt.Errorf("analytics GA4 returned status %d", response.StatusCode)
	}

	ga4SendCount.Add(ctx, 1)
	ga4SendDuration.Record(ctx, duration)

	return nil
}

// releaseSnapshotParams returns all pooled params maps from a batch
// back to the pool.
//
// Takes batch ([]ga4EventSnapshot) which holds the snapshots to release.
func (gc *GA4Collector) releaseSnapshotParams(batch []ga4EventSnapshot) {
	for index := range batch {
		if batch[index].params != nil {
			gc.releaseParams(batch[index].params)
			batch[index].params = nil
		}
	}
}

// acquireParams returns a cleared map from the pool, or allocates a
// new one on pool miss.
//
// Returns map[string]any which is the reusable parameter map.
func (gc *GA4Collector) acquireParams() map[string]any {
	if pooled, ok := gc.paramsPool.Get().(map[string]any); ok {
		clear(pooled)
		return pooled
	}
	return make(map[string]any, ga4ParamsInitialCapacity)
}

// releaseParams returns a params map to the pool.
//
// Takes params (map[string]any) which is the map to return.
func (gc *GA4Collector) releaseParams(params map[string]any) {
	gc.paramsPool.Put(params)
}

// sha256Pool reuses SHA-256 hashers to avoid a 96-byte allocation
// on every Collect call.
var sha256Pool = sync.Pool{
	New: func() any { return sha256.New() },
}

// separatorByte is the delimiter written between client IP and user
// agent when computing the client ID hash.
var separatorByte = []byte("|")

// defaultGA4ClientID produces a deterministic pseudo-anonymous
// client identifier by hashing the client IP and user agent.
// The hasher is pooled to avoid per-call allocation.
//
// Takes clientIP (string) which is the client's IP address.
// Takes userAgent (string) which is the client's user agent.
//
// Returns string which is the hex-encoded SHA-256 hash.
func defaultGA4ClientID(clientIP, userAgent string) string {
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
