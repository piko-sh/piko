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
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"sync"
	"time"

	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/retry"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

const (
	// defaultWebhookBatchSize is the number of events per batch.
	defaultWebhookBatchSize = 10

	// defaultWebhookFlushInterval is the time between automatic flushes.
	defaultWebhookFlushInterval = 5 * time.Second

	// defaultWebhookTimeout is the HTTP client timeout.
	defaultWebhookTimeout = 10 * time.Second

	// webhookCollectorName identifies this collector in logs and metrics.
	webhookCollectorName = "webhook"

	// httpStatusErrorThreshold is the lowest HTTP status code treated as
	// an error response from the webhook endpoint.
	httpStatusErrorThreshold = 400

	// maxResponseDiscardSize is the upper bound when draining an HTTP
	// response body to enable connection reuse (64 KiB).
	maxResponseDiscardSize = 64 << 10

	// maxPooledBufferCapacity is the largest buffer capacity kept in
	// the pool. Buffers that grew beyond this threshold during a
	// spike are discarded to avoid lasting memory bloat.
	maxPooledBufferCapacity = 256 << 10
)

// jsonBufferPool provides reusable bytes.Buffer instances for JSON
// encoding, avoiding allocation on every batch send.
var jsonBufferPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// eventSnapshot is a serialisable copy of event data. The raw
// *http.Request is not retained.
type eventSnapshot struct {
	// Revenue holds optional monetary data for e-commerce events.
	// Nil when the event does not carry revenue information.
	Revenue *maths.Money `json:"revenue,omitempty"`

	// Properties holds arbitrary key-value metadata for the event.
	Properties map[string]string `json:"properties,omitempty"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Hostname is the request host (e.g. "example.com").
	Hostname string `json:"hostname,omitempty"`

	// URL is the full request URL including query parameters.
	URL string `json:"url,omitempty"`

	// ClientIP is the real client IP as resolved by the RealIP middleware.
	ClientIP string `json:"client_ip"`

	// Path is the URL path of the request.
	Path string `json:"path"`

	// Method is the HTTP method (GET, POST, etc.).
	Method string `json:"method"`

	// UserAgent is the User-Agent header value.
	UserAgent string `json:"user_agent"`

	// Referrer is the Referer header value.
	Referrer string `json:"referrer,omitempty"`

	// MatchedPattern is the route pattern that matched the request.
	MatchedPattern string `json:"matched_pattern,omitempty"`

	// Locale is the request locale (e.g. "en", "de").
	Locale string `json:"locale,omitempty"`

	// UserID is the authenticated user's identifier, empty if anonymous.
	UserID string `json:"user_id,omitempty"`

	// ActionName is the name of the server action, empty for page views.
	ActionName string `json:"action_name,omitempty"`

	// EventName is the explicit custom event name (e.g. "signup").
	EventName string `json:"event_name,omitempty"`

	// Type is the event classification ("pageview", "action", "custom").
	Type string `json:"type"`

	// DurationMS is the request handling duration in milliseconds.
	DurationMS float64 `json:"duration_ms"`

	// StatusCode is the HTTP response status code.
	StatusCode int `json:"status_code"`
}

// WebhookOption configures a WebhookCollector.
type WebhookOption func(*WebhookCollector)

// WithWebhookHeaders sets custom HTTP headers sent with each batch
// POST (e.g. Authorization).
//
// Takes headers (http.Header) which are merged into every request.
//
// Returns WebhookOption which configures the headers.
func WithWebhookHeaders(headers http.Header) WebhookOption {
	return func(wc *WebhookCollector) {
		wc.headers = headers.Clone()
	}
}

// WithWebhookBatchSize sets the maximum number of events per batch.
// Defaults to 10.
//
// Takes size (int) which is the batch capacity.
//
// Returns WebhookOption which configures the batch size.
func WithWebhookBatchSize(size int) WebhookOption {
	return func(wc *WebhookCollector) {
		if size <= 0 {
			log.Warn("WithWebhookBatchSize ignored non-positive value",
				logger_domain.Int("size", size))
			return
		}
		wc.batchSize = size
	}
}

// WithWebhookFlushInterval sets the time between automatic batch
// flushes. Defaults to 5 seconds.
//
// Takes d (time.Duration) which is the flush interval.
//
// Returns WebhookOption which configures the interval.
func WithWebhookFlushInterval(d time.Duration) WebhookOption {
	return func(wc *WebhookCollector) {
		if d <= 0 {
			log.Warn("WithWebhookFlushInterval ignored non-positive value",
				logger_domain.String("duration", d.String()))
			return
		}
		wc.flushInterval = d
	}
}

// WithWebhookTimeout sets the HTTP client timeout for batch POSTs.
// Defaults to 10 seconds.
//
// Takes d (time.Duration) which is the client timeout.
//
// Returns WebhookOption which configures the timeout.
func WithWebhookTimeout(d time.Duration) WebhookOption {
	return func(wc *WebhookCollector) {
		if d <= 0 {
			log.Warn("WithWebhookTimeout ignored non-positive value",
				logger_domain.String("duration", d.String()))
			return
		}
		wc.client.Timeout = d
	}
}

// WithWebhookRetry enables retry with exponential backoff for
// failed batch sends. Only retryable errors (network failures,
// 5xx) are retried; permanent errors fail immediately.
//
// Takes config (retry.Config) which configures the retry behaviour.
//
// Returns WebhookOption which configures the retry.
func WithWebhookRetry(config retry.Config) WebhookOption {
	return func(wc *WebhookCollector) {
		wc.retryConfig = &config
	}
}

// WithWebhookCircuitBreaker enables a circuit breaker that stops
// sending batches after consecutive failures. The circuit reopens
// after the timeout expires and a probe request succeeds.
//
// Takes config (analytics_domain.CircuitBreakerConfig) which
// configures the circuit breaker.
//
// Returns WebhookOption which configures the circuit breaker.
func WithWebhookCircuitBreaker(config analytics_domain.CircuitBreakerConfig) WebhookOption {
	return func(wc *WebhookCollector) {
		wc.circuitBreakerConfig = &config
	}
}

// withWebhookClock sets the clock used by the batcher for
// timer-based flushes.
//
// Takes c (clock.Clock) which provides time operations.
//
// Returns WebhookOption which configures the clock.
func withWebhookClock(c clock.Clock) WebhookOption {
	return func(wc *WebhookCollector) {
		wc.clock = c
	}
}

// WebhookCollector posts analytics events as JSON batches to a
// configurable URL. Events are buffered internally and flushed when
// the batch reaches batchSize or the flushInterval expires.
type WebhookCollector struct {
	// batcher manages the buffer, flush loop, and lifecycle.
	batcher *analytics_domain.Batcher[eventSnapshot]

	// client is the HTTP client used for batch POST requests.
	client *http.Client

	// headers holds custom HTTP headers sent with each batch POST
	// (e.g. Authorization).
	headers http.Header

	// retryConfig holds optional retry settings. Nil disables retry.
	retryConfig *retry.Config

	// circuitBreakerConfig holds optional circuit breaker settings.
	// Nil disables the circuit breaker.
	circuitBreakerConfig *analytics_domain.CircuitBreakerConfig

	// clock provides time operations for the batcher. Nil defaults
	// to clock.RealClock() in the batcher.
	clock clock.Clock

	// url is the webhook endpoint that receives JSON event batches.
	url string

	// flushInterval is the time between automatic timer-based flushes.
	flushInterval time.Duration

	// batchSize is the maximum number of events per batch before an
	// immediate flush is triggered.
	batchSize int
}

// NewWebhookCollector creates a collector that POSTs JSON batches to
// the given URL.
//
// Takes endpoint (string) which is the webhook endpoint.
// Takes opts (...WebhookOption) which configure the collector.
//
// Returns *WebhookCollector which is the configured collector.
// Returns error when the endpoint or configuration is invalid.
func NewWebhookCollector(endpoint string, opts ...WebhookOption) (*WebhookCollector, error) {
	if endpoint == "" {
		return nil, errors.New("analytics: webhook URL must not be empty")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("analytics: invalid webhook URL %q: %w", endpoint, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("analytics: webhook URL %q must be an absolute URL with scheme and host", endpoint)
	}

	wc := &WebhookCollector{
		client:        &http.Client{Timeout: defaultWebhookTimeout},
		url:           endpoint,
		batchSize:     defaultWebhookBatchSize,
		flushInterval: defaultWebhookFlushInterval,
	}
	for _, opt := range opts {
		opt(wc)
	}

	batcher, err := analytics_domain.NewBatcher[eventSnapshot](
		analytics_domain.BatcherConfig{
			Name:           webhookCollectorName,
			BatchSize:      wc.batchSize,
			FlushInterval:  wc.flushInterval,
			Clock:          wc.clock,
			Retry:          wc.retryConfig,
			CircuitBreaker: wc.circuitBreakerConfig,
		},
		wc.sendBatch,
	)
	if err != nil {
		return nil, fmt.Errorf("analytics: creating webhook batcher: %w", err)
	}
	wc.batcher = batcher
	return wc, nil
}

// Start launches the background flush loop. Called by the analytics
// Service after registration.
func (wc *WebhookCollector) Start(ctx context.Context) {
	wc.batcher.Start(ctx)
}

// Collect copies the event data into the internal buffer.
//
// When the buffer reaches batchSize the flush goroutine is signalled
// to send the batch asynchronously; Collect itself never performs I/O.
//
// Takes event (*analytics_dto.Event) which carries the event data.
//
// Returns error which is always nil.
func (wc *WebhookCollector) Collect(_ context.Context, event *analytics_dto.Event) error {
	snap := eventSnapshot{
		Hostname:       event.Hostname,
		URL:            event.URL,
		ClientIP:       event.ClientIP,
		Path:           event.Path,
		Method:         event.Method,
		UserAgent:      event.UserAgent,
		Referrer:       event.Referrer,
		MatchedPattern: event.MatchedPattern,
		Locale:         event.Locale,
		UserID:         event.UserID,
		ActionName:     event.ActionName,
		EventName:      event.EventName,
		Timestamp:      event.Timestamp,
		DurationMS:     float64(event.Duration) / float64(time.Millisecond),
		StatusCode:     event.StatusCode,
		Type:           event.Type.String(),
	}
	if event.Revenue != nil {
		snap.Revenue = new(*event.Revenue)
	}
	if event.Properties != nil {
		snap.Properties = make(map[string]string, len(event.Properties))
		maps.Copy(snap.Properties, event.Properties)
	}

	wc.batcher.Add(snap)
	return nil
}

// Flush sends any buffered events to the webhook endpoint.
//
// Returns error when the POST fails.
func (wc *WebhookCollector) Flush(ctx context.Context) error {
	return wc.batcher.Flush(ctx)
}

// Close stops the flush timer and releases resources.
//
// Any remaining buffered events should be flushed via Flush before
// calling Close. Safe to call multiple times.
//
// Returns error which is always nil.
func (wc *WebhookCollector) Close(_ context.Context) error {
	return wc.batcher.Close()
}

// Name returns the collector name.
//
// Returns string which identifies this collector.
func (*WebhookCollector) Name() string {
	return webhookCollectorName
}

// sendBatch encodes and POSTs a batch of event snapshots to the
// webhook endpoint.
//
// Takes batch ([]eventSnapshot) which holds the events to send.
//
// Returns error when encoding or the HTTP request fails.
func (wc *WebhookCollector) sendBatch(ctx context.Context, batch []eventSnapshot) (returnErr error) {
	defer func() {
		if r := recover(); r != nil {
			returnErr = goroutine.HandlePanicRecovery(ctx, "analytics_adapters.webhook.sendBatch", r)
		}
	}()

	ctx, l := logger_domain.From(ctx, log)
	webhookBatchSize.Record(ctx, int64(len(batch)))

	body, err := encodeBatch(batch)
	if err != nil {
		webhookErrorCount.Add(ctx, 1)
		l.Warn("Analytics webhook JSON encoding failed", logger_domain.Error(err))
		return err
	}

	return wc.postBatch(ctx, body)
}

// encodeBatch serialises the batch into a pooled buffer.
//
// Takes batch ([]eventSnapshot) which holds the events to encode.
//
// Returns []byte which is the encoded JSON payload.
// Returns error when encoding fails.
func encodeBatch(batch []eventSnapshot) ([]byte, error) {
	buf, ok := jsonBufferPool.Get().(*bytes.Buffer)
	if !ok {
		buf = new(bytes.Buffer)
	}
	buf.Reset()
	defer func() {
		if buf.Cap() <= maxPooledBufferCapacity {
			jsonBufferPool.Put(buf)
		}
	}()

	if err := json.NewEncoder(buf).Encode(batch); err != nil {
		return nil, fmt.Errorf("encoding analytics batch: %w", err)
	}
	return bytes.Clone(buf.Bytes()), nil
}

// postBatch sends the encoded payload to the webhook endpoint.
//
// Takes body ([]byte) which is the JSON payload.
//
// Returns error when the HTTP request fails or returns an error
// status.
func (wc *WebhookCollector) postBatch(ctx context.Context, body []byte) error {
	ctx, l := logger_domain.From(ctx, log)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, wc.url, bytes.NewReader(body))
	if err != nil {
		webhookErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating analytics webhook request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	for key, values := range wc.headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}

	start := time.Now()
	response, err := wc.client.Do(request)
	duration := float64(time.Since(start)) / float64(time.Millisecond)

	if err != nil {
		webhookErrorCount.Add(ctx, 1)
		l.Warn("Analytics webhook POST failed", logger_domain.Error(err))
		return fmt.Errorf("posting analytics batch: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxResponseDiscardSize))
		_ = response.Body.Close()
	}()

	webhookSendCount.Add(ctx, 1)
	webhookSendDuration.Record(ctx, duration)

	if response.StatusCode >= httpStatusErrorThreshold {
		webhookErrorCount.Add(ctx, 1)
		l.Warn("Analytics webhook returned error status",
			logger_domain.Int("status_code", response.StatusCode))
		return fmt.Errorf("analytics webhook returned status %d", response.StatusCode)
	}

	return nil
}
