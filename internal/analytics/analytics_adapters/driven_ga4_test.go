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
	"piko.sh/piko/wdk/maths"
)

func TestGA4Collector_BatchFlush(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var payload ga4Payload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("unmarshalling GA4 body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL, WithGA4BatchSize(3))

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
		if err := gc.Collect(context.Background(), event); err != nil {
			t.Fatalf("Collect returned error: %v", err)
		}
	}

	if err := gc.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := gc.Close(context.Background()); err != nil {
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

func TestGA4Collector_PageViewMapping(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload ga4Payload
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL, WithGA4BatchSize(1))

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
	_ = gc.Collect(context.Background(), event)
	_ = gc.Flush(context.Background())
	_ = gc.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 || len(received[0].Events) != 1 {
		t.Fatalf("expected 1 payload with 1 event")
	}
	ga4Event := received[0].Events[0]
	if ga4Event.Name != "page_view" {
		t.Errorf("name = %q, want page_view", ga4Event.Name)
	}
	if ga4Event.Params["page_location"] != "https://example.com/products?cat=shoes" {
		t.Errorf("page_location = %v, want full URL", ga4Event.Params["page_location"])
	}
	if ga4Event.Params["page_referrer"] != "https://google.com" {
		t.Errorf("page_referrer = %v", ga4Event.Params["page_referrer"])
	}
	if ga4Event.Params["language"] != "en-GB" {
		t.Errorf("language = %v, want en-GB", ga4Event.Params["language"])
	}
}

func TestGA4Collector_CustomEventName(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload ga4Payload
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL, WithGA4BatchSize(1))

	event := &analytics_dto.Event{
		EventName:  "purchase",
		Path:       "/checkout",
		StatusCode: 200,
		Type:       analytics_dto.EventCustom,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = gc.Collect(context.Background(), event)
	_ = gc.Flush(context.Background())
	_ = gc.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].Events[0].Name != "purchase" {
		t.Errorf("name = %q, want purchase", received[0].Events[0].Name)
	}
}

func TestGA4Collector_ActionNameMapping(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload ga4Payload
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL, WithGA4BatchSize(1))

	event := &analytics_dto.Event{
		ActionName: "cart.Purchase",
		Path:       "/api/action",
		StatusCode: 200,
		Type:       analytics_dto.EventAction,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = gc.Collect(context.Background(), event)
	_ = gc.Flush(context.Background())
	_ = gc.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].Events[0].Name != "cart.Purchase" {
		t.Errorf("name = %q, want cart.Purchase", received[0].Events[0].Name)
	}
	if received[0].Events[0].Params["action_name"] != "cart.Purchase" {
		t.Errorf("action_name param = %v", received[0].Events[0].Params["action_name"])
	}
}

func TestGA4Collector_RevenueMapping(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload ga4Payload
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL, WithGA4BatchSize(1))

	revenue := maths.NewMoneyFromString("49.99", "GBP")
	event := &analytics_dto.Event{
		EventName:  "purchase",
		Path:       "/checkout",
		StatusCode: 200,
		Type:       analytics_dto.EventCustom,
		Revenue:    &revenue,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = gc.Collect(context.Background(), event)
	_ = gc.Flush(context.Background())
	_ = gc.Close(context.Background())

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

func TestGA4Collector_PropertiesMerge(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload ga4Payload
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL, WithGA4BatchSize(1))

	event := &analytics_dto.Event{
		Path:       "/signup",
		StatusCode: 200,
		Type:       analytics_dto.EventCustom,
		Properties: map[string]string{"plan": "pro", "source": "organic"},
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = gc.Collect(context.Background(), event)
	_ = gc.Flush(context.Background())
	_ = gc.Close(context.Background())

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

func TestGA4Collector_ClientIDFromIPAndUA(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload ga4Payload
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL, WithGA4BatchSize(1))

	event := &analytics_dto.Event{
		Path:       "/test",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "192.168.1.1",
		UserAgent:  "TestBrowser/1.0",
	}
	_ = gc.Collect(context.Background(), event)
	_ = gc.Flush(context.Background())
	_ = gc.Close(context.Background())

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

func TestGA4Collector_CustomClientIDFunc(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload ga4Payload
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL,
		WithGA4BatchSize(1),
		WithGA4ClientIDFunc(func(_, _ string) string { return "custom-client-123" }),
	)

	event := &analytics_dto.Event{
		Path:       "/test",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = gc.Collect(context.Background(), event)
	_ = gc.Flush(context.Background())
	_ = gc.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].ClientID != "custom-client-123" {
		t.Errorf("client_id = %q, want custom-client-123", received[0].ClientID)
	}
}

func TestGA4Collector_UserIDMapping(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload ga4Payload
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL, WithGA4BatchSize(1))

	event := &analytics_dto.Event{
		Path:       "/account",
		StatusCode: 200,
		UserID:     "user-alex-demo",
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = gc.Collect(context.Background(), event)
	_ = gc.Flush(context.Background())
	_ = gc.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].UserID != "user-alex-demo" {
		t.Errorf("user_id = %q, want user-alex-demo", received[0].UserID)
	}
}

func TestGA4Collector_TimestampMicros(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload ga4Payload
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL, WithGA4BatchSize(1))

	eventTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	event := &analytics_dto.Event{
		Path:       "/test",
		StatusCode: 200,
		Timestamp:  eventTime,
		ClientIP:   "10.0.0.1",
	}
	_ = gc.Collect(context.Background(), event)
	_ = gc.Flush(context.Background())
	_ = gc.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if received[0].TimestampMicros != eventTime.UnixMicro() {
		t.Errorf("timestamp_micros = %d, want %d", received[0].TimestampMicros, eventTime.UnixMicro())
	}
}

func TestGA4Collector_DebugEndpoint(t *testing.T) {
	var receivedURL string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gc := NewGA4Collector("G-TEST", "secret", WithGA4Debug(true))
	if !strings.Contains(gc.endpoint, "/debug/mp/collect") {
		t.Errorf("endpoint = %q, want /debug/mp/collect", gc.endpoint)
	}
	_ = gc.Close(context.Background())

	gcProd := NewGA4Collector("G-TEST", "secret")
	if !strings.Contains(gcProd.endpoint, "/mp/collect") {
		t.Errorf("endpoint = %q, want /mp/collect", gcProd.endpoint)
	}
	if strings.Contains(gcProd.endpoint, "/debug/") {
		t.Errorf("production endpoint should not contain /debug/")
	}
	_ = gcProd.Close(context.Background())

	_ = receivedURL
}

func TestGA4Collector_FlushEmptyBuffer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected POST to GA4 with empty buffer")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL)

	if err := gc.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := gc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestGA4Collector_Name(t *testing.T) {
	gc := NewGA4Collector("G-TEST", "secret")
	defer gc.Close(context.Background())

	if gc.Name() != "ga4" {
		t.Errorf("Name() = %q, want ga4", gc.Name())
	}
}

func TestGA4Collector_DoubleClose(t *testing.T) {
	gc := NewGA4Collector("G-TEST", "secret")

	if err := gc.Close(context.Background()); err != nil {
		t.Fatalf("first Close returned error: %v", err)
	}
	if err := gc.Close(context.Background()); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
}

func TestGA4Collector_ErrorStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL, WithGA4BatchSize(1))

	event := &analytics_dto.Event{
		Path:       "/error",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = gc.Collect(context.Background(), event)
	err := gc.Flush(context.Background())
	_ = gc.Close(context.Background())

	if err == nil {
		t.Error("expected error from Flush when server returns 500")
	}
}

func TestGA4Collector_TimerFlush(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload ga4Payload
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL,
		WithGA4BatchSize(100),
		WithGA4FlushInterval(50*time.Millisecond),
	)

	event := &analytics_dto.Event{
		Path:       "/timer",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = gc.Collect(context.Background(), event)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := len(received)
	mu.Unlock()

	if count == 0 {
		t.Error("expected timer-based flush to send events")
	}
	_ = gc.Close(context.Background())
}

func TestGA4Collector_MaxBatchClamped(t *testing.T) {
	gc := NewGA4Collector("G-TEST", "secret", WithGA4BatchSize(50))
	defer gc.Close(context.Background())

	if gc.batchSize != maxGA4EventsPerRequest {
		t.Errorf("batchSize = %d, want %d (clamped)", gc.batchSize, maxGA4EventsPerRequest)
	}
}

func TestGA4Collector_ClientIDPartitioning(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload ga4Payload
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL,
		WithGA4BatchSize(10),
		WithGA4FlushInterval(1*time.Hour),
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
	_ = gc.Collect(context.Background(), event1)
	_ = gc.Collect(context.Background(), event2)
	_ = gc.Flush(context.Background())
	_ = gc.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 2 {
		t.Fatalf("expected 2 payloads (one per client_id), got %d", len(received))
	}
	if received[0].ClientID == received[1].ClientID {
		t.Errorf("expected different client_ids, both are %q", received[0].ClientID)
	}
}

func TestGA4Collector_AnonymousClientID(t *testing.T) {
	clientID := defaultGA4ClientID("", "")
	if clientID != "anonymous" {
		t.Errorf("defaultGA4ClientID('', '') = %q, want anonymous", clientID)
	}
}

func TestGA4Collector_FlushOnShutdown(t *testing.T) {
	var mu sync.Mutex
	var received []ga4Payload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload ga4Payload
		_ = json.Unmarshal(body, &payload)
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	gc := newTestGA4Collector(srv.URL,
		WithGA4BatchSize(100),
		WithGA4FlushInterval(1*time.Hour),
	)

	event := &analytics_dto.Event{
		Path:       "/pending",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}
	_ = gc.Collect(context.Background(), event)
	_ = gc.Flush(context.Background())
	_ = gc.Close(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 payload after Flush, got %d", len(received))
	}
}

func TestGA4Collector_EndpointQueryParams(t *testing.T) {
	gc := NewGA4Collector("G-ABCDEF", "my-secret-key")
	defer gc.Close(context.Background())

	if !strings.Contains(gc.endpoint, "measurement_id=G-ABCDEF") {
		t.Errorf("endpoint missing measurement_id: %s", gc.endpoint)
	}
	if !strings.Contains(gc.endpoint, "api_secret=my-secret-key") {
		t.Errorf("endpoint missing api_secret: %s", gc.endpoint)
	}
}

func newTestGA4Collector(testURL string, opts ...GA4Option) *GA4Collector {
	allOpts := append([]GA4Option{WithGA4FlushInterval(1 * time.Hour)}, opts...)
	gc := NewGA4Collector("G-TEST", "test-secret", allOpts...)
	gc.endpoint = testURL
	gc.Start(context.Background())
	return gc
}
