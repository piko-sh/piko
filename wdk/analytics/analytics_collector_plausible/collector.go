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

package analytics_collector_plausible

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	"piko.sh/piko/wdk/clock"
)

const (
	// defaultBatchSize is the number of events per batch.
	defaultBatchSize = 10

	// defaultFlushInterval is the time between automatic flushes.
	defaultFlushInterval = 5 * time.Second

	// defaultTimeout is the HTTP client timeout.
	defaultTimeout = 10 * time.Second

	// collectorName identifies this collector in logs and metrics.
	collectorName = "plausible"

	// defaultEndpoint is the Plausible API base URL.
	defaultEndpoint = "https://plausible.io"

	// eventPath is the Plausible Events API path.
	eventPath = "/api/event"

	// maxProps is the maximum number of custom properties per event.
	maxProps = 30

	// maxURLLength is the maximum URL length before truncation.
	maxURLLength = 2000

	// httpStatusErrorThreshold is the lowest HTTP status code
	// treated as an error.
	httpStatusErrorThreshold = 400

	// maxResponseDiscardSize is the upper bound when draining an
	// HTTP response body to enable connection reuse (64 KiB).
	maxResponseDiscardSize = 64 << 10

	// maxPooledBufferCapacity is the largest buffer capacity kept in
	// the pool; buffers that grew beyond this during a spike are
	// discarded to avoid lasting memory bloat.
	maxPooledBufferCapacity = 256 << 10
)

// jsonBufferPool provides reusable bytes.Buffer instances for JSON
// encoding, avoiding allocation on every event send.
var jsonBufferPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// eventPayload is the JSON body sent to the Plausible Events API.
type eventPayload struct {
	// Revenue holds optional monetary data for e-commerce events.
	Revenue *revenuePayload `json:"revenue,omitempty"`

	// Props holds custom event properties (max 30 key-value pairs).
	Props map[string]string `json:"props,omitempty"`

	// Interactive controls whether the event affects bounce rate.
	// Plausible defaults to true; set to false for background
	// actions that should not count as user engagement.
	Interactive *bool `json:"interactive,omitempty"`

	// Domain is the site domain registered in Plausible.
	Domain string `json:"domain"`

	// Name is the event name ("pageview" or a custom event name).
	Name string `json:"name"`

	// URL is the page URL where the event occurred.
	URL string `json:"url"`

	// Referrer is the referring page URL.
	Referrer string `json:"referrer,omitempty"`
}

// revenuePayload is the Plausible revenue tracking structure.
type revenuePayload struct {
	// Currency is an ISO 4217 currency code (e.g. "USD", "GBP").
	Currency string `json:"currency"`

	// Amount is the monetary value as a string (e.g. "29.99").
	Amount string `json:"amount"`
}

// snapshot carries the JSON payload plus per-request headers that
// cannot be included in the body.
type snapshot struct {
	// payload is the JSON body to send.
	payload eventPayload

	// userAgent is set as the User-Agent header (required by
	// Plausible for device detection).
	userAgent string

	// clientIP is set as the X-Forwarded-For header for geo and
	// unique visitor identification.
	clientIP string
}

// Option configures a Plausible [Collector].
type Option func(*Collector)

// WithEndpoint sets the Plausible API base URL for self-hosted
// instances. Defaults to "https://plausible.io".
//
// Takes url (string) which is the base URL (without /api/event).
//
// Returns Option which configures the endpoint.
func WithEndpoint(endpointURL string) Option {
	return func(c *Collector) {
		if endpointURL == "" {
			log.Warn("WithEndpoint ignored empty URL")
			return
		}
		c.endpoint = strings.TrimRight(endpointURL, "/")
	}
}

// WithBatchSize sets the number of events buffered before triggering
// a flush. Defaults to 10.
//
// Takes size (int) which is the batch capacity.
//
// Returns Option which configures the batch size.
func WithBatchSize(size int) Option {
	return func(c *Collector) {
		if size <= 0 {
			log.Warn("WithBatchSize ignored non-positive value",
				logger_domain.Int("size", size))
			return
		}
		c.batchSize = size
	}
}

// WithFlushInterval sets the time between automatic batch flushes.
// Defaults to 5 seconds.
//
// Takes d (time.Duration) which is the flush interval.
//
// Returns Option which configures the interval.
func WithFlushInterval(d time.Duration) Option {
	return func(c *Collector) {
		if d <= 0 {
			log.Warn("WithFlushInterval ignored non-positive value",
				logger_domain.String("duration", d.String()))
			return
		}
		c.flushInterval = d
	}
}

// WithTimeout sets the HTTP client timeout for event POSTs.
// Defaults to 10 seconds.
//
// Takes d (time.Duration) which is the client timeout.
//
// Returns Option which configures the timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Collector) {
		if d <= 0 {
			log.Warn("WithTimeout ignored non-positive value",
				logger_domain.String("duration", d.String()))
			return
		}
		c.client.Timeout = d
	}
}

// WithRetry enables retry with exponential backoff for failed event
// sends. Only retryable errors (network failures, 5xx) are retried;
// permanent errors fail immediately.
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

// WithCircuitBreaker enables a circuit breaker that stops sending
// events after consecutive failures. The circuit reopens after the
// timeout expires and a probe request succeeds.
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

// withClock sets the clock used by the batcher for timer-based
// flushes.
//
// Takes c (clock.Clock) which provides time operations.
//
// Returns Option which configures the clock.
func withClock(c clock.Clock) Option {
	return func(collector *Collector) {
		collector.clock = c
	}
}

// Collector sends analytics events to the Plausible Analytics Events
// API.
//
// Events are buffered internally and flushed when the batch reaches
// batchSize or the flushInterval expires. Each event is sent as a
// separate HTTP POST (Plausible has no batch endpoint).
type Collector struct {
	// batcher manages the buffer, flush loop, and lifecycle.
	batcher *analytics_domain.Batcher[snapshot]

	// client is the HTTP client used for event POST requests.
	client *http.Client

	// retryConfig holds optional retry settings.
	retryConfig *retry.Config

	// circuitBreakerConfig holds optional circuit breaker settings.
	circuitBreakerConfig *analytics_domain.CircuitBreakerConfig

	// propsPool recycles map[string]string instances to avoid
	// per-event allocation in Collect.
	propsPool sync.Pool

	// domain is the site domain registered in Plausible.
	domain string

	// endpoint is the Plausible API base URL.
	endpoint string

	// clock provides time operations for the batcher. Nil defaults
	// to clock.RealClock() in the batcher.
	clock clock.Clock

	// flushInterval is the time between automatic flushes.
	flushInterval time.Duration

	// batchSize is the number of events that triggers a flush.
	batchSize int
}

// NewCollector creates an analytics collector that sends events to
// the Plausible Analytics Events API.
//
// Takes domain (string) which is the site domain registered in
// your Plausible account.
// Takes opts (...Option) which configure the collector.
//
// Returns analytics.Collector which sends events to Plausible.
// Returns error when the domain or configuration is invalid.
func NewCollector(domain string, opts ...Option) (analytics.Collector, error) {
	if domain == "" {
		return nil, errors.New("analytics: Plausible domain must not be empty")
	}

	c := &Collector{
		client:        &http.Client{Timeout: defaultTimeout},
		domain:        domain,
		endpoint:      defaultEndpoint,
		batchSize:     defaultBatchSize,
		flushInterval: defaultFlushInterval,
	}
	for _, opt := range opts {
		opt(c)
	}

	batcher, err := analytics_domain.NewBatcher[snapshot](
		analytics_domain.BatcherConfig{
			Name:           collectorName,
			BatchSize:      c.batchSize,
			FlushInterval:  c.flushInterval,
			Clock:          c.clock,
			Retry:          c.retryConfig,
			CircuitBreaker: c.circuitBreakerConfig,
		},
		c.sendBatch,
	)
	if err != nil {
		return nil, fmt.Errorf("analytics: creating Plausible batcher: %w", err)
	}
	c.batcher = batcher
	return c, nil
}

// Start launches the background flush loop.
func (c *Collector) Start(ctx context.Context) {
	c.batcher.Start(ctx)
}

// Collect copies the event data into the internal buffer. Collect
// itself never performs I/O.
//
// Takes event (*analytics_dto.Event) which carries the event data.
//
// Returns error which is always nil.
func (c *Collector) Collect(_ context.Context, event *analytics_dto.Event) error {
	snap := snapshot{
		userAgent: event.UserAgent,
		clientIP:  event.ClientIP,
		payload: eventPayload{
			Domain: c.resolveDomain(event),
			Name:   resolveEventName(event),
			URL:    resolveURL(event),
		},
	}

	if event.Referrer != "" {
		snap.payload.Referrer = event.Referrer
	}

	if event.Type == analytics_dto.EventAction && event.EventName == "" {
		snap.payload.Interactive = new(false)
	}

	if event.Revenue != nil {
		revenueValue := *event.Revenue
		if currencyCode, currencyError := revenueValue.CurrencyCode(); currencyError == nil {
			if number, numberError := revenueValue.Number(); numberError == nil {
				snap.payload.Revenue = &revenuePayload{
					Currency: currencyCode,
					Amount:   number,
				}
			} else {
				log.Warn("Analytics Plausible revenue number extraction failed",
					logger_domain.Error(numberError))
			}
		} else {
			log.Warn("Analytics Plausible revenue currency code extraction failed",
				logger_domain.Error(currencyError))
		}
	}

	if event.Properties != nil {
		props := c.acquireProps()
		count := 0
		for key, value := range event.Properties {
			if count >= maxProps {
				break
			}
			props[key] = value
			count++
		}
		snap.payload.Props = props
	}

	c.batcher.Add(snap)
	return nil
}

// Flush sends any buffered events to Plausible.
//
// Returns error when any event POST fails.
func (c *Collector) Flush(ctx context.Context) error {
	return c.batcher.Flush(ctx)
}

// Close stops the flush timer and releases resources. Safe to call
// multiple times.
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

// HealthCheck verifies that the Plausible batcher is running and its
// circuit breaker is not open.
//
// Returns error when the collector is unhealthy.
func (c *Collector) HealthCheck(_ context.Context) error {
	return c.batcher.HealthCheck()
}

// resolveDomain returns the event's hostname if set, falling back
// to the configured domain.
//
// Takes event (*analytics_dto.Event) which may carry a hostname.
//
// Returns string which is the domain for the Plausible payload.
func (c *Collector) resolveDomain(event *analytics_dto.Event) string {
	if event.Hostname != "" {
		return event.Hostname
	}
	return c.domain
}

// resolveEventName maps the analytics event type to a Plausible
// event name.
//
// Takes event (*analytics_dto.Event) which carries the event data.
//
// Returns string which is the Plausible event name.
func resolveEventName(event *analytics_dto.Event) string {
	switch {
	case event.EventName != "":
		return event.EventName
	case event.Type == analytics_dto.EventPageView:
		return "pageview"
	case event.Type == analytics_dto.EventAction:
		if event.ActionName != "" {
			return event.ActionName
		}
		return "action"
	default:
		return "custom"
	}
}

// resolveURL constructs an absolute URL for the Plausible payload,
// truncated to maxURLLength.
//
// Takes event (*analytics_dto.Event) which carries URL data.
//
// Returns string which is the absolute URL.
func resolveURL(event *analytics_dto.Event) string {
	var url string
	switch {
	case event.URL != "" && strings.HasPrefix(event.URL, "http"):
		url = event.URL
	case event.Hostname != "" && event.URL != "":
		url = "https://" + event.Hostname + event.URL
	case event.Hostname != "" && event.Path != "":
		url = "https://" + event.Hostname + event.Path
	case event.URL != "":
		url = event.URL
	default:
		url = "/"
	}

	if len(url) > maxURLLength {
		url = url[:maxURLLength]
	}
	return url
}

// sendBatch iterates through the batch and sends each event
// individually (Plausible has no batch endpoint).
//
// Takes batch ([]snapshot) which holds the events to send.
//
// Returns error which aggregates failures from all event sends.
func (c *Collector) sendBatch(ctx context.Context, batch []snapshot) error {
	batchSize.Record(ctx, int64(len(batch)))

	defer c.releaseSnapshotProps(batch)

	var errs []error
	for index := range batch {
		if ctx.Err() != nil {
			errs = append(errs, ctx.Err())
			break
		}

		if err := c.sendEvent(ctx, &batch[index]); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// sendEvent encodes and POSTs a single event to the Plausible API.
//
// Takes snap (*snapshot) which holds the event data.
//
// Returns error when encoding or the HTTP request fails.
func (c *Collector) sendEvent(ctx context.Context, snap *snapshot) (returnErr error) {
	defer func() {
		if r := recover(); r != nil {
			returnErr = goroutine.HandlePanicRecovery(ctx, "analytics_collector_plausible.sendEvent", r)
		}
	}()

	ctx, l := logger_domain.From(ctx, log)
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

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(snap.payload); err != nil {
		errorCount.Add(ctx, 1)
		l.Warn("Analytics Plausible JSON encoding failed", logger_domain.Error(err))
		return fmt.Errorf("encoding analytics Plausible event: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.endpoint+eventPath, bytes.NewReader(buf.Bytes()))
	if err != nil {
		errorCount.Add(ctx, 1)
		return fmt.Errorf("creating analytics Plausible request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", snap.userAgent)
	if snap.clientIP != "" {
		request.Header.Set("X-Forwarded-For", snap.clientIP)
	}

	start := time.Now()
	response, err := c.client.Do(request)
	duration := float64(time.Since(start)) / float64(time.Millisecond)

	if err != nil {
		errorCount.Add(ctx, 1)
		l.Warn("Analytics Plausible POST failed", logger_domain.Error(err))
		return fmt.Errorf("posting analytics Plausible event: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxResponseDiscardSize))
		_ = response.Body.Close()
	}()

	if response.StatusCode >= httpStatusErrorThreshold {
		errorCount.Add(ctx, 1)
		l.Warn("Analytics Plausible returned error status",
			logger_domain.Int("status_code", response.StatusCode))
		return fmt.Errorf("analytics Plausible returned status %d", response.StatusCode)
	}

	sendCount.Add(ctx, 1)
	sendDuration.Record(ctx, duration)

	return nil
}

// acquireProps returns a cleared map from the pool, or allocates a
// new one on pool miss.
//
// Returns map[string]string which is the reusable properties map.
func (c *Collector) acquireProps() map[string]string {
	if pooled, ok := c.propsPool.Get().(map[string]string); ok {
		clear(pooled)
		return pooled
	}
	return make(map[string]string, maxProps)
}

// releaseProps returns a properties map to the pool.
//
// Takes props (map[string]string) which is the map to return.
func (c *Collector) releaseProps(props map[string]string) {
	c.propsPool.Put(props)
}

// releaseSnapshotProps returns all pooled props maps from a batch
// back to the pool.
//
// Takes batch ([]snapshot) which holds the snapshots to release.
func (c *Collector) releaseSnapshotProps(batch []snapshot) {
	for index := range batch {
		if batch[index].payload.Props != nil {
			c.releaseProps(batch[index].payload.Props)
			batch[index].payload.Props = nil
		}
	}
}
