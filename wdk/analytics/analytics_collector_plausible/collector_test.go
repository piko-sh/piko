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
	"piko.sh/piko/wdk/maths"
)

func TestCollector_PageView(t *testing.T) {
	var mu sync.Mutex
	var received []receivedEvent

	srv := newTestServer(&mu, &received)
	defer srv.Close()

	collector := newTestCollector(srv.URL)

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

	collector := newTestCollector(srv.URL)

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

	collector := newTestCollector(srv.URL)

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

	collector := newTestCollector(srv.URL)

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

	collector := newTestCollector(srv.URL)

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

	collector := newTestCollector(srv.URL)

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

	collector := newTestCollector(srv.URL)

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

	collector := newTestCollector(srv.URL)

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

func TestCollector_EmptyDomainPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on empty domain")
		}
	}()
	NewCollector("")
}

func TestCollector_FlushEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected request with empty buffer")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	collector := newTestCollector(srv.URL)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())
}

func TestCollector_ErrorStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	collector := newTestCollector(srv.URL)

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
	collector := NewCollector("example.com").(*Collector) //nolint:revive // test-only assertion
	defer collector.Close(context.Background())

	if collector.Name() != "plausible" {
		t.Errorf("Name() = %q, want plausible", collector.Name())
	}
}

func TestCollector_DoubleClose(t *testing.T) {
	collector := NewCollector("example.com").(*Collector) //nolint:revive // test-only assertion

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

	collector := newTestCollector(srv.URL)

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

	collector := newTestCollector(srv.URL)

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

	collector := newTestCollector(srv.URL)

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

// receivedEvent captures the HTTP request details for assertions.
type receivedEvent struct {
	payload      eventPayload
	userAgent    string
	forwardedFor string
}

// newTestServer creates a mock Plausible API server that captures
// received events.
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

// newTestCollector creates a Plausible collector pointing at a test
// server URL.
func newTestCollector(testURL string, opts ...Option) *Collector {
	allOpts := append([]Option{WithFlushInterval(1 * time.Hour)}, opts...)
	collector := NewCollector("test.example.com", allOpts...).(*Collector) //nolint:revive // test-only assertion
	collector.endpoint = testURL
	collector.Start(context.Background())
	return collector
}
