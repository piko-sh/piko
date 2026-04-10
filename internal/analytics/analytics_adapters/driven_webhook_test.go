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
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/wdk/maths"
)

func TestWebhookCollector_BatchFlush(t *testing.T) {
	var mu sync.Mutex
	var received [][]eventSnapshot

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var batch []eventSnapshot
		if err := json.Unmarshal(body, &batch); err != nil {
			t.Errorf("unmarshalling body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, batch)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wc := NewWebhookCollector(srv.URL,
		WithWebhookBatchSize(3),
		WithWebhookFlushInterval(1*time.Hour),
	)

	for i := range 3 {
		ev := &analytics_dto.Event{
			Path:       "/page",
			Method:     http.MethodGet,
			StatusCode: 200,
			Type:       analytics_dto.EventPageView,
			Timestamp:  time.Now(),
		}
		_ = i
		if err := wc.Collect(context.Background(), ev); err != nil {
			t.Fatalf("Collect returned error: %v", err)
		}
	}

	if err := wc.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := wc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	mu.Lock()
	count := len(received)
	mu.Unlock()
	if count != 1 {
		t.Fatalf("expected 1 batch POST, got %d", count)
	}

	mu.Lock()
	batch := received[0]
	mu.Unlock()
	if len(batch) != 3 {
		t.Fatalf("expected 3 events in batch, got %d", len(batch))
	}
	if batch[0].Path != "/page" {
		t.Errorf("Path = %q, want /page", batch[0].Path)
	}
	if batch[0].Type != "pageview" {
		t.Errorf("Type = %q, want pageview", batch[0].Type)
	}
}

func TestWebhookCollector_FlushOnShutdown(t *testing.T) {
	var mu sync.Mutex
	var received [][]eventSnapshot

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var batch []eventSnapshot
		if err := json.Unmarshal(body, &batch); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, batch)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wc := NewWebhookCollector(srv.URL,
		WithWebhookBatchSize(100),
		WithWebhookFlushInterval(1*time.Hour),
	)

	ev := &analytics_dto.Event{
		Path:       "/pending",
		StatusCode: 200,
		Type:       analytics_dto.EventCustom,
		Timestamp:  time.Now(),
	}
	if err := wc.Collect(context.Background(), ev); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if err := wc.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := wc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	mu.Lock()
	count := len(received)
	mu.Unlock()
	if count != 1 {
		t.Fatalf("expected 1 batch POST after Flush, got %d", count)
	}

	mu.Lock()
	if received[0][0].Type != "custom" {
		t.Errorf("Type = %q, want custom", received[0][0].Type)
	}
	mu.Unlock()
}

func TestWebhookCollector_CustomHeaders(t *testing.T) {
	var receivedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	headers := http.Header{}
	headers.Set("Authorization", "Bearer test-token")

	wc := NewWebhookCollector(srv.URL,
		WithWebhookBatchSize(1),
		WithWebhookHeaders(headers),
	)

	ev := &analytics_dto.Event{
		Path:       "/auth",
		StatusCode: 200,
		Timestamp:  time.Now(),
	}
	if err := wc.Collect(context.Background(), ev); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if err := wc.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := wc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	if receivedAuth != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want Bearer test-token", receivedAuth)
	}
}

func TestWebhookCollector_Properties(t *testing.T) {
	var mu sync.Mutex
	var received [][]eventSnapshot

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var batch []eventSnapshot
		if err := json.Unmarshal(body, &batch); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, batch)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wc := NewWebhookCollector(srv.URL, WithWebhookBatchSize(1))

	ev := &analytics_dto.Event{
		Path:       "/purchase",
		StatusCode: 200,
		Type:       analytics_dto.EventCustom,
		ActionName: "checkout",
		Properties: map[string]string{"plan": "pro", "amount": "29.99"},
		Timestamp:  time.Now(),
	}
	if err := wc.Collect(context.Background(), ev); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if err := wc.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := wc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 || len(received[0]) != 1 {
		t.Fatalf("expected 1 batch with 1 event")
	}
	snap := received[0][0]
	if snap.ActionName != "checkout" {
		t.Errorf("ActionName = %q, want checkout", snap.ActionName)
	}
	if snap.Properties["plan"] != "pro" {
		t.Errorf("Properties[plan] = %q, want pro", snap.Properties["plan"])
	}
	if snap.Properties["amount"] != "29.99" {
		t.Errorf("Properties[amount] = %q, want 29.99", snap.Properties["amount"])
	}
}

func TestWebhookCollector_WithTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wc := NewWebhookCollector(srv.URL,
		WithWebhookBatchSize(1),
		WithWebhookTimeout(30*time.Second),
	)

	ev := &analytics_dto.Event{
		Path:       "/timeout-test",
		StatusCode: 200,
		Timestamp:  time.Now(),
	}
	if err := wc.Collect(context.Background(), ev); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if err := wc.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := wc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestWebhookCollector_ErrorStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	wc := NewWebhookCollector(srv.URL, WithWebhookBatchSize(1))

	ev := &analytics_dto.Event{
		Path:       "/error",
		StatusCode: 200,
		Timestamp:  time.Now(),
	}
	if err := wc.Collect(context.Background(), ev); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	_ = wc.Flush(context.Background())
	if err := wc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestWebhookCollector_TimerFlush(t *testing.T) {
	var mu sync.Mutex
	var received [][]eventSnapshot

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var batch []eventSnapshot
		if err := json.Unmarshal(body, &batch); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, batch)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wc := NewWebhookCollector(srv.URL,
		WithWebhookBatchSize(100),
		WithWebhookFlushInterval(50*time.Millisecond),
	)

	ev := &analytics_dto.Event{
		Path:       "/timer",
		StatusCode: 200,
		Timestamp:  time.Now(),
	}
	if err := wc.Collect(context.Background(), ev); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := len(received)
	mu.Unlock()

	if count == 0 {
		t.Error("expected timer-based flush to send events")
	}

	if err := wc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestWebhookCollector_FlushEmptyBuffer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected POST to webhook with empty buffer")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wc := NewWebhookCollector(srv.URL)

	if err := wc.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := wc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestWebhookCollector_Name(t *testing.T) {
	wc := NewWebhookCollector("http://localhost")
	defer wc.Close(context.Background())

	if wc.Name() != "webhook" {
		t.Errorf("Name() = %q, want webhook", wc.Name())
	}
}

func TestWebhookCollector_DoubleClose(t *testing.T) {
	wc := NewWebhookCollector("http://localhost")

	if err := wc.Close(context.Background()); err != nil {
		t.Fatalf("first Close returned error: %v", err)
	}

	if err := wc.Close(context.Background()); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
}

func TestWebhookCollector_RevenueAndNewFields(t *testing.T) {
	var mu sync.Mutex
	var received [][]eventSnapshot

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var batch []eventSnapshot
		if err := json.Unmarshal(body, &batch); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, batch)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wc := NewWebhookCollector(srv.URL, WithWebhookBatchSize(1))

	rev := maths.NewMoneyFromString("49.99", "GBP")
	ev := &analytics_dto.Event{
		Hostname:   "shop.example.com",
		URL:        "/checkout?ref=email",
		Path:       "/checkout",
		EventName:  "purchase",
		StatusCode: 200,
		Type:       analytics_dto.EventCustom,
		Revenue:    &rev,
		Timestamp:  time.Now(),
	}
	if err := wc.Collect(context.Background(), ev); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if err := wc.Flush(context.Background()); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if err := wc.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 || len(received[0]) != 1 {
		t.Fatalf("expected 1 batch with 1 event")
	}
	snap := received[0][0]
	if snap.Hostname != "shop.example.com" {
		t.Errorf("Hostname = %q, want shop.example.com", snap.Hostname)
	}
	if snap.URL != "/checkout?ref=email" {
		t.Errorf("URL = %q, want /checkout?ref=email", snap.URL)
	}
	if snap.EventName != "purchase" {
		t.Errorf("EventName = %q, want purchase", snap.EventName)
	}
	if snap.Revenue == nil {
		t.Fatal("Revenue is nil, want non-nil")
	}
	revenueNumber := snap.Revenue.MustNumber()
	if revenueNumber != "49.99" {
		t.Errorf("Revenue amount = %q, want 49.99", revenueNumber)
	}
	currencyCode, err := snap.Revenue.CurrencyCode()
	if err != nil {
		t.Fatalf("Revenue.CurrencyCode() error: %v", err)
	}
	if currencyCode != "GBP" {
		t.Errorf("Revenue currency = %q, want GBP", currencyCode)
	}
	if snap.Type != "custom" {
		t.Errorf("Type = %q, want custom", snap.Type)
	}
}
