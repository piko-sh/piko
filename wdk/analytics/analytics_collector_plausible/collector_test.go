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
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/wdk/analytics"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

func TestCollector_PageView(t *testing.T) {
	var mu sync.Mutex
	var received []receivedEvent

	srv := newTestServer(&mu, &received)
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	event := &analytics_dto.Event{
		Hostname:  "example.com",
		URL:       "/products?page=2",
		Path:      "/products",
		Referrer:  "https://google.com",
		UserAgent: "Mozilla/5.0",
		ClientIP:  "192.168.1.1",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}
	ev := received[0]
	if ev.payload.Domain != "example.com" {
		t.Errorf("domain = %q, want example.com", ev.payload.Domain)
	}
	if ev.payload.Name != "pageview" {
		t.Errorf("name = %q, want pageview", ev.payload.Name)
	}
	if ev.payload.URL != "https://example.com/products?page=2" {
		t.Errorf("url = %q, want https://example.com/products?page=2", ev.payload.URL)
	}
	if ev.payload.Referrer != "https://google.com" {
		t.Errorf("referrer = %q, want https://google.com", ev.payload.Referrer)
	}
	if ev.userAgent != "Mozilla/5.0" {
		t.Errorf("User-Agent = %q, want Mozilla/5.0", ev.userAgent)
	}
	if ev.forwardedFor != "192.168.1.1" {
		t.Errorf("X-Forwarded-For = %q, want 192.168.1.1", ev.forwardedFor)
	}
}

func TestCollector_CustomEvent(t *testing.T) {
	var mu sync.Mutex
	var received []receivedEvent

	srv := newTestServer(&mu, &received)
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	event := &analytics_dto.Event{
		Hostname:  "example.com",
		URL:       "/signup",
		EventName: "signup",
		UserAgent: "Bot",
		Type:      analytics_dto.EventCustom,
		Timestamp: time.Now(),
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].payload.Name != "signup" {
		t.Errorf("name = %q, want signup", received[0].payload.Name)
	}
}

func TestCollector_ActionEvent(t *testing.T) {
	var mu sync.Mutex
	var received []receivedEvent

	srv := newTestServer(&mu, &received)
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	event := &analytics_dto.Event{
		Hostname:   "example.com",
		URL:        "/api/action",
		ActionName: "cart.Purchase",
		UserAgent:  "Bot",
		Type:       analytics_dto.EventAction,
		Timestamp:  time.Now(),
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].payload.Name != "cart.Purchase" {
		t.Errorf("name = %q, want cart.Purchase", received[0].payload.Name)
	}
}

func TestCollector_Revenue(t *testing.T) {
	var mu sync.Mutex
	var received []receivedEvent

	srv := newTestServer(&mu, &received)
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	revenue := maths.NewMoneyFromString("29.99", "GBP")
	event := &analytics_dto.Event{
		Hostname:  "shop.example.com",
		URL:       "/checkout",
		EventName: "purchase",
		UserAgent: "Bot",
		Revenue:   &revenue,
		Type:      analytics_dto.EventCustom,
		Timestamp: time.Now(),
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].payload.Revenue == nil {
		t.Fatal("revenue is nil")
	}
	if received[0].payload.Revenue.Currency != "GBP" {
		t.Errorf("currency = %q, want GBP", received[0].payload.Revenue.Currency)
	}
	if received[0].payload.Revenue.Amount != "29.99" {
		t.Errorf("amount = %q, want 29.99", received[0].payload.Revenue.Amount)
	}
}

func TestCollector_Properties(t *testing.T) {
	var mu sync.Mutex
	var received []receivedEvent

	srv := newTestServer(&mu, &received)
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	event := &analytics_dto.Event{
		Hostname:   "example.com",
		URL:        "/pricing",
		UserAgent:  "Bot",
		Properties: map[string]string{"plan": "pro", "source": "organic"},
		Type:       analytics_dto.EventPageView,
		Timestamp:  time.Now(),
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].payload.Props["plan"] != "pro" {
		t.Errorf("props[plan] = %q, want pro", received[0].payload.Props["plan"])
	}
	if received[0].payload.Props["source"] != "organic" {
		t.Errorf("props[source] = %q, want organic", received[0].payload.Props["source"])
	}
}

func TestCollector_PropsLimitedTo30(t *testing.T) {
	var mu sync.Mutex
	var received []receivedEvent

	srv := newTestServer(&mu, &received)
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	props := make(map[string]string, 35)
	for i := range 35 {
		props[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("val_%d", i)
	}

	event := &analytics_dto.Event{
		Hostname:   "example.com",
		URL:        "/test",
		UserAgent:  "Bot",
		Properties: props,
		Type:       analytics_dto.EventPageView,
		Timestamp:  time.Now(),
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if len(received[0].payload.Props) > maxProps {
		t.Errorf("props count = %d, want <= %d", len(received[0].payload.Props), maxProps)
	}
}

func TestCollector_URLTruncation(t *testing.T) {
	var mu sync.Mutex
	var received []receivedEvent

	srv := newTestServer(&mu, &received)
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	longPath := "/" + strings.Repeat("a", maxURLLength+100)
	event := &analytics_dto.Event{
		Hostname:  "example.com",
		URL:       longPath,
		UserAgent: "Bot",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if len(received[0].payload.URL) > maxURLLength {
		t.Errorf("url length = %d, want <= %d", len(received[0].payload.URL), maxURLLength)
	}
}

func TestCollector_SelfHostedEndpoint(t *testing.T) {
	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	event := &analytics_dto.Event{
		Hostname:  "example.com",
		URL:       "/test",
		UserAgent: "Bot",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	if requestCount.Load() != 1 {
		t.Errorf("expected 1 request to custom endpoint, got %d", requestCount.Load())
	}
}

func TestCollector_EmptyDomainReturnsError(t *testing.T) {
	_, err := NewCollector("")
	if err == nil {
		t.Fatal("expected error on empty domain")
	}
	if !strings.Contains(err.Error(), "domain must not be empty") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCollector_FlushEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected request with empty buffer")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())
}

func TestCollector_ErrorStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	event := &analytics_dto.Event{
		Hostname:  "example.com",
		URL:       "/test",
		UserAgent: "Bot",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}
	_ = collector.Collect(context.Background(), event)
	err := collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	if err == nil {
		t.Error("expected error from 400 response")
	}
}

func TestCollector_Name(t *testing.T) {
	result, err := NewCollector("example.com")
	if err != nil {
		t.Fatalf("NewCollector: %v", err)
	}
	collector := result.(*Collector)
	defer collector.Close(context.Background())

	if collector.Name() != "plausible" {
		t.Errorf("Name() = %q, want plausible", collector.Name())
	}
}

func TestCollector_DoubleClose(t *testing.T) {
	result, err := NewCollector("example.com")
	if err != nil {
		t.Fatalf("NewCollector: %v", err)
	}
	collector := result.(*Collector)

	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestCollector_MultipleEvents(t *testing.T) {
	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	for range 3 {
		event := &analytics_dto.Event{
			Hostname:  "example.com",
			URL:       "/page",
			UserAgent: "Bot",
			Type:      analytics_dto.EventPageView,
			Timestamp: time.Now(),
		}
		_ = collector.Collect(context.Background(), event)
	}
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	if requestCount.Load() != 3 {
		t.Errorf("expected 3 separate requests, got %d", requestCount.Load())
	}
}

func TestCollector_NoClientIP(t *testing.T) {
	var mu sync.Mutex
	var received []receivedEvent

	srv := newTestServer(&mu, &received)
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	event := &analytics_dto.Event{
		Hostname:  "example.com",
		URL:       "/test",
		UserAgent: "Bot",
		ClientIP:  "",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].forwardedFor != "" {
		t.Errorf("X-Forwarded-For = %q, want empty", received[0].forwardedFor)
	}
}

func TestCollector_FallbackDomain(t *testing.T) {
	var mu sync.Mutex
	var received []receivedEvent

	srv := newTestServer(&mu, &received)
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	event := &analytics_dto.Event{
		URL:       "/test",
		UserAgent: "Bot",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].payload.Domain != "test.example.com" {
		t.Errorf("domain = %q, want test.example.com (fallback)", received[0].payload.Domain)
	}
}

func TestResolveEventName(t *testing.T) {
	tests := []struct {
		name     string
		event    *analytics_dto.Event
		expected string
	}{
		{
			name:     "explicit event name takes priority",
			event:    &analytics_dto.Event{EventName: "signup"},
			expected: "signup",
		},
		{
			name:     "page view type returns pageview",
			event:    &analytics_dto.Event{Type: analytics_dto.EventPageView},
			expected: "pageview",
		},
		{
			name: "action type with action name",
			event: &analytics_dto.Event{
				Type:       analytics_dto.EventAction,
				ActionName: "cart.Purchase",
			},
			expected: "cart.Purchase",
		},
		{
			name:     "action type without action name falls back",
			event:    &analytics_dto.Event{Type: analytics_dto.EventAction},
			expected: "action",
		},
		{
			name:     "custom type returns custom",
			event:    &analytics_dto.Event{Type: analytics_dto.EventCustom},
			expected: "custom",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := resolveEventName(testCase.event)
			if got != testCase.expected {
				t.Errorf("resolveEventName() = %q, want %q", got, testCase.expected)
			}
		})
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name     string
		event    *analytics_dto.Event
		expected string
	}{
		{
			name:     "absolute URL used as-is",
			event:    &analytics_dto.Event{URL: "https://example.com/page"},
			expected: "https://example.com/page",
		},
		{
			name: "hostname plus URL constructs absolute URL",
			event: &analytics_dto.Event{
				Hostname: "example.com",
				URL:      "/products?page=2",
			},
			expected: "https://example.com/products?page=2",
		},
		{
			name: "hostname plus path constructs absolute URL",
			event: &analytics_dto.Event{
				Hostname: "example.com",
				Path:     "/about",
			},
			expected: "https://example.com/about",
		},
		{
			name:     "relative URL without hostname used as-is",
			event:    &analytics_dto.Event{URL: "/local-path"},
			expected: "/local-path",
		},
		{
			name:     "all empty returns root",
			event:    &analytics_dto.Event{},
			expected: "/",
		},
		{
			name: "truncation to maxURLLength",
			event: &analytics_dto.Event{
				URL: "https://example.com/" + strings.Repeat("x", maxURLLength),
			},
			expected: ("https://example.com/" + strings.Repeat("x", maxURLLength))[:maxURLLength],
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := resolveURL(testCase.event)
			if got != testCase.expected {
				t.Errorf("resolveURL() = %q, want %q", got, testCase.expected)
			}
		})
	}
}

func TestWithTimeout(t *testing.T) {
	result, err := NewCollector("example.com",
		WithTimeout(42*time.Second),
	)
	if err != nil {
		t.Fatalf("NewCollector: %v", err)
	}
	collector := result.(*Collector)
	defer collector.Close(context.Background())

	if collector.client.Timeout != 42*time.Second {
		t.Errorf("client timeout = %v, want 42s", collector.client.Timeout)
	}
}

func TestWithRetry(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := attempts.Add(1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL, WithRetry(analytics.RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Millisecond,
		MaxDelay:      5 * time.Millisecond,
		BackoffFactor: 2.0,
		JitterFunc:    func(d time.Duration) time.Duration { return 0 },
	}))

	event := &analytics_dto.Event{
		Hostname:  "example.com",
		URL:       "/retry-test",
		UserAgent: "Bot",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}
	_ = collector.Collect(context.Background(), event)
	err := collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	if err != nil {
		t.Fatalf("expected successful retry, got error: %v", err)
	}
	if attempts.Load() < 3 {
		t.Errorf("expected at least 3 attempts (2 failures + 1 success), got %d", attempts.Load())
	}
}

func TestWithClock(t *testing.T) {
	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	result, err := NewCollector("example.com",
		withClock(mockClock),
	)
	if err != nil {
		t.Fatalf("NewCollector: %v", err)
	}
	collector := result.(*Collector)
	defer collector.Close(context.Background())

	if collector.clock != mockClock {
		t.Error("expected collector clock to be the injected mock clock")
	}
}

type receivedEvent struct {
	payload      eventPayload
	userAgent    string
	forwardedFor string
}

func newTestServer(mu *sync.Mutex, received *[]receivedEvent) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var payload eventPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		*received = append(*received, receivedEvent{
			payload:      payload,
			userAgent:    r.Header.Get("User-Agent"),
			forwardedFor: r.Header.Get("X-Forwarded-For"),
		})
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
}

func newTestCollector(t *testing.T, testURL string, opts ...Option) *Collector {
	t.Helper()

	allOpts := append([]Option{WithFlushInterval(1 * time.Hour)}, opts...)
	result, err := NewCollector("test.example.com", allOpts...)
	if err != nil {
		t.Fatalf("newTestCollector: %v", err)
	}
	collector := result.(*Collector)
	collector.endpoint = testURL
	collector.Start(context.Background())
	return collector
}
