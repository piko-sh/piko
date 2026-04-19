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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

func TestCollector_BatchFlush(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL, WithBatchSize(3))

	for range 3 {
		event := &analytics_dto.Event{
			Path:       "/page",
			Method:     http.MethodGet,
			StatusCode: 200,
			Type:       analytics_dto.EventPageView,
			Timestamp:  time.Now(),
			ClientIP:   "1.2.3.4",
			UserAgent:  "TestBot",
		}
		if err := collector.Collect(context.Background(), event); err != nil {
			t.Fatalf("Collect returned error: %v", err)
		}
	}

	if err := collector.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 batch POST, got %d", len(received))
	}
	if len(received[0].Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(received[0].Events))
	}
	if received[0].Events[0].Name != "page_view" {
		t.Errorf("event name = %q, want page_view", received[0].Events[0].Name)
	}
}

func TestCollector_PageViewMapping(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL, WithBatchSize(1))

	event := &analytics_dto.Event{
		Hostname:   "example.com",
		URL:        "https://example.com/products?cat=shoes",
		Path:       "/products",
		Referrer:   "https://google.com",
		Locale:     "en-GB",
		Method:     http.MethodGet,
		StatusCode: 200,
		Type:       analytics_dto.EventPageView,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
		UserAgent:  "Mozilla/5.0",
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 || len(received[0].Events) != 1 {
		t.Fatalf("expected 1 payload with 1 event")
	}
	ev := received[0].Events[0]
	if ev.Name != "page_view" {
		t.Errorf("name = %q, want page_view", ev.Name)
	}
	if ev.Params["page_location"] != "https://example.com/products?cat=shoes" {
		t.Errorf("page_location = %v, want full URL", ev.Params["page_location"])
	}
	if ev.Params["page_referrer"] != "https://google.com" {
		t.Errorf("page_referrer = %v", ev.Params["page_referrer"])
	}
	if ev.Params["language"] != "en-GB" {
		t.Errorf("language = %v, want en-GB", ev.Params["language"])
	}
}

func TestCollector_CustomEventName(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL, WithBatchSize(1))

	event := &analytics_dto.Event{
		EventName:  "purchase",
		Path:       "/checkout",
		StatusCode: 200,
		Type:       analytics_dto.EventCustom,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].Events[0].Name != "purchase" {
		t.Errorf("name = %q, want purchase", received[0].Events[0].Name)
	}
}

func TestCollector_ActionNameMapping(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL, WithBatchSize(1))

	event := &analytics_dto.Event{
		ActionName: "cart.Purchase",
		Path:       "/api/action",
		StatusCode: 200,
		Type:       analytics_dto.EventAction,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].Events[0].Name != "cart.Purchase" {
		t.Errorf("name = %q, want cart.Purchase", received[0].Events[0].Name)
	}
	if received[0].Events[0].Params["action_name"] != "cart.Purchase" {
		t.Errorf("action_name param = %v", received[0].Events[0].Params["action_name"])
	}
}

func TestCollector_RevenueMapping(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL, WithBatchSize(1))

	event := &analytics_dto.Event{
		EventName:  "purchase",
		Path:       "/checkout",
		StatusCode: 200,
		Type:       analytics_dto.EventCustom,
		Revenue:    new(maths.NewMoneyFromString("49.99", "GBP")),
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	params := received[0].Events[0].Params
	if params["currency"] != "GBP" {
		t.Errorf("currency = %v, want GBP", params["currency"])
	}
	value, ok := params["value"].(float64)
	if !ok {
		t.Fatalf("value is not float64: %T", params["value"])
	}
	if value != 49.99 {
		t.Errorf("value = %v, want 49.99", value)
	}
}

func TestCollector_PropertiesMerge(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL, WithBatchSize(1))

	event := &analytics_dto.Event{
		Path:       "/signup",
		StatusCode: 200,
		Type:       analytics_dto.EventCustom,
		Properties: map[string]string{"plan": "pro", "source": "organic"},
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	params := received[0].Events[0].Params
	if params["plan"] != "pro" {
		t.Errorf("plan = %v, want pro", params["plan"])
	}
	if params["source"] != "organic" {
		t.Errorf("source = %v, want organic", params["source"])
	}
}

func TestCollector_ClientIDFromIPAndUA(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL, WithBatchSize(1))

	event := &analytics_dto.Event{
		Path:       "/test",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "192.168.1.1",
		UserAgent:  "TestBrowser/1.0",
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	expectedHash := sha256.New()
	expectedHash.Write([]byte("192.168.1.1"))
	expectedHash.Write([]byte("|"))
	expectedHash.Write([]byte("TestBrowser/1.0"))
	expectedClientID := hex.EncodeToString(expectedHash.Sum(nil))

	mu.Lock()
	defer mu.Unlock()
	if received[0].ClientID != expectedClientID {
		t.Errorf("client_id = %q, want %q", received[0].ClientID, expectedClientID)
	}
}

func TestCollector_CustomClientIDFunc(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL,
		WithBatchSize(1),
		WithClientIDFunc(func(_, _ string) string { return "custom-client-123" }),
	)

	event := &analytics_dto.Event{
		Path:       "/test",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].ClientID != "custom-client-123" {
		t.Errorf("client_id = %q, want custom-client-123", received[0].ClientID)
	}
}

func TestCollector_UserIDMapping(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL, WithBatchSize(1))

	event := &analytics_dto.Event{
		Path:       "/account",
		StatusCode: 200,
		UserID:     "user-alex-demo",
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].UserID != "user-alex-demo" {
		t.Errorf("user_id = %q, want user-alex-demo", received[0].UserID)
	}
}

func TestCollector_TimestampMicros(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL, WithBatchSize(1))

	eventTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	event := &analytics_dto.Event{
		Path:       "/test",
		StatusCode: 200,
		Timestamp:  eventTime,
		ClientIP:   "10.0.0.1",
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].TimestampMicros != eventTime.UnixMicro() {
		t.Errorf("timestamp_micros = %d, want %d", received[0].TimestampMicros, eventTime.UnixMicro())
	}
}

func TestCollector_DebugEndpoint(t *testing.T) {
	var receivedURL string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	result, err := NewCollector("G-TEST", "secret", WithDebug(true))
	if err != nil {
		t.Fatalf("NewCollector returned unexpected error: %v", err)
	}
	collector := result.(*Collector)
	if !strings.Contains(collector.endpoint, "/debug/mp/collect") {
		t.Errorf("endpoint = %q, want /debug/mp/collect", collector.endpoint)
	}
	_ = collector.Close(context.Background())

	resultProd, err := NewCollector("G-TEST", "secret")
	if err != nil {
		t.Fatalf("NewCollector returned unexpected error: %v", err)
	}
	collectorProd := resultProd.(*Collector)
	if !strings.Contains(collectorProd.endpoint, "/mp/collect") {
		t.Errorf("endpoint = %q, want /mp/collect", collectorProd.endpoint)
	}
	if strings.Contains(collectorProd.endpoint, "/debug/") {
		t.Errorf("production endpoint should not contain /debug/")
	}
	_ = collectorProd.Close(context.Background())

	_ = receivedURL
}

func TestCollector_FlushEmptyBuffer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected POST to GA4 with empty buffer")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL)

	if err := collector.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestCollector_Name(t *testing.T) {
	result, err := NewCollector("G-TEST", "secret")
	if err != nil {
		t.Fatalf("NewCollector returned unexpected error: %v", err)
	}
	collector := result.(*Collector)
	defer collector.Close(context.Background())

	if collector.Name() != "ga4" {
		t.Errorf("Name() = %q, want ga4", collector.Name())
	}
}

func TestCollector_DoubleClose(t *testing.T) {
	result, err := NewCollector("G-TEST", "secret")
	if err != nil {
		t.Fatalf("NewCollector returned unexpected error: %v", err)
	}
	collector := result.(*Collector)

	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("first Close returned error: %v", err)
	}
	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
}

func TestCollector_ErrorStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL, WithBatchSize(1))

	event := &analytics_dto.Event{
		Path:       "/error",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = collector.Collect(context.Background(), event)
	err := collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	if err == nil {
		t.Error("expected error from Flush when server returns 500")
	}
}

func TestCollector_TimerFlush(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	mockClock := clock.NewMockClock(time.Now())
	collector := newTestCollector(t, srv.URL,
		WithBatchSize(100),
		WithFlushInterval(5*time.Second),
		withClock(mockClock),
	)

	event := &analytics_dto.Event{
		Path:       "/timer",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = collector.Collect(context.Background(), event)

	mockClock.Advance(6 * time.Second)

	if err := collector.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}

	mu.Lock()
	count := len(received)
	mu.Unlock()

	if count == 0 {
		t.Error("expected timer-based flush to send events")
	}
	_ = collector.Close(context.Background())
}

func TestCollector_MaxBatchClamped(t *testing.T) {
	result, err := NewCollector("G-TEST", "secret", WithBatchSize(50))
	if err != nil {
		t.Fatalf("NewCollector returned unexpected error: %v", err)
	}
	collector := result.(*Collector)
	defer collector.Close(context.Background())

	if collector.batchSize != maxEventsPerRequest {
		t.Errorf("batchSize = %d, want %d (clamped)", collector.batchSize, maxEventsPerRequest)
	}
}

func TestCollector_ClientIDPartitioning(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL,
		WithBatchSize(10),
		WithFlushInterval(1*time.Hour),
	)

	event1 := &analytics_dto.Event{
		Path: "/page1", StatusCode: 200, Timestamp: time.Now(),
		ClientIP: "1.1.1.1", UserAgent: "Bot",
		Type: analytics_dto.EventPageView,
	}
	event2 := &analytics_dto.Event{
		Path: "/page2", StatusCode: 200, Timestamp: time.Now(),
		ClientIP: "2.2.2.2", UserAgent: "Bot",
		Type: analytics_dto.EventPageView,
	}
	_ = collector.Collect(context.Background(), event1)
	_ = collector.Collect(context.Background(), event2)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 2 {
		t.Fatalf("expected 2 payloads (one per client_id), got %d", len(received))
	}
	if received[0].ClientID == received[1].ClientID {
		t.Errorf("expected different client_ids, both are %q", received[0].ClientID)
	}
}

func TestCollector_AnonymousClientID(t *testing.T) {
	clientID := defaultClientID("", "")
	if clientID != "anonymous" {
		t.Errorf("defaultClientID('', '') = %q, want anonymous", clientID)
	}
}

func TestCollector_FlushOnShutdown(t *testing.T) {
	var mu sync.Mutex
	var received []payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if err := json.Unmarshal(body, &p); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	collector := newTestCollector(t, srv.URL,
		WithBatchSize(100),
		WithFlushInterval(1*time.Hour),
	)

	event := &analytics_dto.Event{
		Path:       "/pending",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = collector.Collect(context.Background(), event)
	_ = collector.Flush(context.Background())
	_ = collector.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 payload after Flush, got %d", len(received))
	}
}

func TestCollector_EndpointQueryParams(t *testing.T) {
	result, err := NewCollector("G-ABCDEF", "my-secret-key")
	if err != nil {
		t.Fatalf("NewCollector returned unexpected error: %v", err)
	}
	collector := result.(*Collector)
	defer collector.Close(context.Background())

	if !strings.Contains(collector.endpoint, "measurement_id=G-ABCDEF") {
		t.Errorf("endpoint missing measurement_id: %s", collector.endpoint)
	}
	if !strings.Contains(collector.endpoint, "api_secret=my-secret-key") {
		t.Errorf("endpoint missing api_secret: %s", collector.endpoint)
	}
}

func TestCollector_EmptyMeasurementIDReturnsError(t *testing.T) {
	_, err := NewCollector("", "secret")
	if err == nil {
		t.Fatal("expected error on empty measurementID")
	}
	if !strings.Contains(err.Error(), "measurementID") {
		t.Errorf("error = %q, want mention of measurementID", err)
	}
}

func TestCollector_EmptyAPISecretReturnsError(t *testing.T) {
	_, err := NewCollector("G-TEST", "")
	if err == nil {
		t.Fatal("expected error on empty apiSecret")
	}
	if !strings.Contains(err.Error(), "apiSecret") {
		t.Errorf("error = %q, want mention of apiSecret", err)
	}
}

func newTestCollector(t *testing.T, testURL string, opts ...Option) *Collector {
	t.Helper()
	allOpts := append([]Option{WithFlushInterval(1 * time.Hour)}, opts...)
	result, err := NewCollector("G-TEST", "test-secret", allOpts...)
	if err != nil {
		t.Fatalf("NewCollector returned unexpected error: %v", err)
	}
	collector := result.(*Collector)
	collector.endpoint = testURL
	collector.Start(context.Background())
	return collector
}
