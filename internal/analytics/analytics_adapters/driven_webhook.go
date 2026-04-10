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
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"

	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
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
)

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
		if size > 0 {
			wc.batchSize = size
		}
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
		if d > 0 {
			wc.flushInterval = d
		}
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
		if d > 0 {
			wc.client.Timeout = d
		}
	}
}

// WebhookCollector posts analytics events as JSON batches to a
// configurable URL. Events are buffered internally and flushed when
// the batch reaches batchSize or the flushInterval expires.
type WebhookCollector struct {
	// client is the HTTP client used for batch POST requests.
	client *http.Client

	// headers holds custom HTTP headers sent with each batch POST
	// (e.g. Authorization).
	headers http.Header

	// stopCh signals the flush goroutine to exit.
	stopCh chan struct{}

	// doneCh is closed when the flush goroutine exits.
	doneCh chan struct{}

	// flushCh is a non-blocking signal that tells the flush goroutine
	// to flush immediately because the buffer reached batchSize.
	flushCh chan struct{}

	// url is the webhook endpoint that receives JSON event batches.
	url string

	// buffer accumulates event snapshots until the batch is flushed.
	buffer []eventSnapshot

	// flushBuffer is a reusable slice for copying the buffer during flush,
	// avoiding a slice allocation per flush cycle.
	flushBuffer []eventSnapshot

	// jsonBuffer is a reusable buffer for JSON encoding, avoiding a byte
	// slice allocation per flush cycle.
	jsonBuffer bytes.Buffer

	// flushInterval is the time between automatic timer-based flushes.
	flushInterval time.Duration

	// batchSize is the maximum number of events per batch before an
	// immediate flush is triggered.
	batchSize int

	// closeOnce ensures Close is idempotent.
	closeOnce sync.Once

	// mu guards buffer access from concurrent Collect and flushLoop calls.
	mu sync.Mutex
}

// NewWebhookCollector creates a collector that POSTs JSON batches to
// the given URL.
//
// Takes url (string) which is the webhook endpoint.
// Takes opts (...WebhookOption) which configure the collector.
//
// Returns *WebhookCollector which is the configured collector.
func NewWebhookCollector(url string, opts ...WebhookOption) *WebhookCollector {
	wc := &WebhookCollector{
		client:        &http.Client{Timeout: defaultWebhookTimeout},
		url:           url,
		batchSize:     defaultWebhookBatchSize,
		flushInterval: defaultWebhookFlushInterval,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
		flushCh:       make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(wc)
	}
	wc.buffer = make([]eventSnapshot, 0, wc.batchSize)
	wc.flushBuffer = make([]eventSnapshot, 0, wc.batchSize)

	go wc.flushLoop()
	return wc
}

// Collect copies the event data into the internal buffer. When the
// buffer reaches batchSize, the flush goroutine is signalled to send
// the batch asynchronously. Collect itself never performs I/O.
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
		rev := *event.Revenue
		snap.Revenue = &rev
	}
	if event.Properties != nil {
		snap.Properties = make(map[string]string, len(event.Properties))
		maps.Copy(snap.Properties, event.Properties)
	}

	wc.mu.Lock()
	wc.buffer = append(wc.buffer, snap)
	full := len(wc.buffer) >= wc.batchSize
	wc.mu.Unlock()

	if full {
		select {
		case wc.flushCh <- struct{}{}:
		default:
		}
	}
	return nil
}

// Flush sends any buffered events to the webhook endpoint.
//
// Returns error when the POST fails.
func (wc *WebhookCollector) Flush(ctx context.Context) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	return wc.flushLocked(ctx)
}

// Close stops the flush timer and releases resources. Any remaining
// buffered events should be flushed via Flush before calling Close.
// Safe to call multiple times.
//
// Returns error which is always nil.
func (wc *WebhookCollector) Close(_ context.Context) error {
	wc.closeOnce.Do(func() {
		close(wc.stopCh)
	})
	<-wc.doneCh
	return nil
}

// Name returns the collector name.
//
// Returns string which identifies this collector.
func (*WebhookCollector) Name() string {
	return webhookCollectorName
}

// flushLoop runs a periodic timer that flushes buffered events. It
// also listens for immediate flush signals when the buffer reaches
// batchSize.
func (wc *WebhookCollector) flushLoop() {
	defer close(wc.doneCh)
	ticker := time.NewTicker(wc.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wc.mu.Lock()
			_ = wc.flushLocked(context.Background())
			wc.mu.Unlock()
		case <-wc.flushCh:
			wc.mu.Lock()
			_ = wc.flushLocked(context.Background())
			wc.mu.Unlock()
		case <-wc.stopCh:
			return
		}
	}
}

// flushLocked sends the current buffer. Caller must hold wc.mu.
// The batch slice and JSON buffer are reused across flushes to avoid
// per-flush allocations.
func (wc *WebhookCollector) flushLocked(ctx context.Context) error {
	if len(wc.buffer) == 0 {
		return nil
	}

	wc.flushBuffer = wc.flushBuffer[:len(wc.buffer)]
	copy(wc.flushBuffer, wc.buffer)
	wc.buffer = wc.buffer[:0]
	batch := wc.flushBuffer

	webhookBatchSize.Record(ctx, int64(len(batch)))

	wc.jsonBuffer.Reset()
	enc := json.NewEncoder(&wc.jsonBuffer)
	if err := enc.Encode(batch); err != nil {
		webhookErrorCount.Add(ctx, 1)
		_, l := logger_domain.From(ctx, log)
		l.Warn("Analytics webhook JSON encoding failed", logger_domain.Error(err))
		return fmt.Errorf("encoding analytics batch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wc.url, bytes.NewReader(wc.jsonBuffer.Bytes()))
	if err != nil {
		webhookErrorCount.Add(ctx, 1)
		return fmt.Errorf("creating analytics webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, vals := range wc.headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	start := time.Now()
	resp, err := wc.client.Do(req)
	duration := float64(time.Since(start)) / float64(time.Millisecond)

	webhookSendCount.Add(ctx, 1)
	webhookSendDuration.Record(ctx, duration)

	if err != nil {
		webhookErrorCount.Add(ctx, 1)
		_, l := logger_domain.From(ctx, log)
		l.Warn("Analytics webhook POST failed", logger_domain.Error(err))
		return fmt.Errorf("posting analytics batch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= httpStatusErrorThreshold {
		webhookErrorCount.Add(ctx, 1)
		_, l := logger_domain.From(ctx, log)
		l.Warn("Analytics webhook returned error status",
			logger_domain.Int("status_code", resp.StatusCode))
		return fmt.Errorf("analytics webhook returned status %d", resp.StatusCode)
	}

	return nil
}
