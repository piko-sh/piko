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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/wdk/analytics"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

type receivedEvent struct {
	payload      eventPayload
	userAgent    string
	forwardedFor string
}

type eventRecorder struct {
	received []receivedEvent
	mu       sync.Mutex
}

func (r *eventRecorder) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if !assert.NoError(t, err, "reading request body") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var payload eventPayload
		if !assert.NoError(t, json.Unmarshal(body, &payload), "unmarshalling Plausible body") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		r.mu.Lock()
		r.received = append(r.received, receivedEvent{
			payload:      payload,
			userAgent:    request.Header.Get("User-Agent"),
			forwardedFor: request.Header.Get("X-Forwarded-For"),
		})
		r.mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}
}

func (r *eventRecorder) events() []receivedEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]receivedEvent(nil), r.received...)
}

func newRecordingServer(t *testing.T) (*httptest.Server, *eventRecorder) {
	t.Helper()
	recorder := &eventRecorder{}
	server := httptest.NewServer(recorder.handler(t))
	t.Cleanup(server.Close)
	return server, recorder
}

func newCountingServer(t *testing.T, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	requestCount := new(atomic.Int32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return server, requestCount
}

func requireCollector(t *testing.T, result any) *Collector {
	t.Helper()
	collector, ok := result.(*Collector)
	require.Truef(t, ok, "NewCollector returned %T, want *Collector", result)
	return collector
}

func newTestCollector(t *testing.T, testURL string, opts ...Option) *Collector {
	t.Helper()
	allOpts := append([]Option{WithFlushInterval(1 * time.Hour)}, opts...)
	result, err := NewCollector("test.example.com", allOpts...)
	require.NoError(t, err)
	collector := requireCollector(t, result)
	collector.endpoint = testURL
	collector.Start(context.Background())
	return collector
}

func collectFlushClose(t *testing.T, collector *Collector, events ...*analytics_dto.Event) error {
	t.Helper()
	ctx := context.Background()
	for _, event := range events {
		require.NoError(t, collector.Collect(ctx, event))
	}
	flushErr := collector.Flush(ctx)
	require.NoError(t, collector.Close(ctx))
	return flushErr
}

func requireSingleEvent(t *testing.T, recorder *eventRecorder) receivedEvent {
	t.Helper()
	received := recorder.events()
	require.Len(t, received, 1)
	return received[0]
}

func TestCollector_PageView(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL)

	require.NoError(t, collectFlushClose(t, collector, &analytics_dto.Event{
		Hostname:  "example.com",
		URL:       "/products?page=2",
		Path:      "/products",
		Referrer:  "https://google.com",
		UserAgent: "Mozilla/5.0",
		ClientIP:  "192.168.1.1",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}))

	event := requireSingleEvent(t, recorder)
	assert.Equal(t, "example.com", event.payload.Domain)
	assert.Equal(t, "pageview", event.payload.Name)
	assert.Equal(t, "https://example.com/products?page=2", event.payload.URL)
	assert.Equal(t, "https://google.com", event.payload.Referrer)
	assert.Equal(t, "Mozilla/5.0", event.userAgent)
	assert.Equal(t, "192.168.1.1", event.forwardedFor)
}

func TestCollector_EventNameMapping(t *testing.T) {
	testCases := []struct {
		event        *analytics_dto.Event
		name         string
		expectedName string
	}{
		{
			name: "custom event name is used",
			event: &analytics_dto.Event{
				Hostname:  "example.com",
				URL:       "/signup",
				EventName: "signup",
				UserAgent: "Bot",
				Type:      analytics_dto.EventCustom,
				Timestamp: time.Now(),
			},
			expectedName: "signup",
		},
		{
			name: "action name becomes the event name",
			event: &analytics_dto.Event{
				Hostname:   "example.com",
				URL:        "/api/action",
				ActionName: "cart.Purchase",
				UserAgent:  "Bot",
				Type:       analytics_dto.EventAction,
				Timestamp:  time.Now(),
			},
			expectedName: "cart.Purchase",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server, recorder := newRecordingServer(t)
			collector := newTestCollector(t, server.URL)

			require.NoError(t, collectFlushClose(t, collector, testCase.event))

			assert.Equal(t, testCase.expectedName, requireSingleEvent(t, recorder).payload.Name)
		})
	}
}

func TestCollector_Revenue(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL)

	revenue := maths.NewMoneyFromString("29.99", "GBP")
	require.NoError(t, collectFlushClose(t, collector, &analytics_dto.Event{
		Hostname:  "shop.example.com",
		URL:       "/checkout",
		EventName: "purchase",
		UserAgent: "Bot",
		Revenue:   &revenue,
		Type:      analytics_dto.EventCustom,
		Timestamp: time.Now(),
	}))

	event := requireSingleEvent(t, recorder)
	require.NotNil(t, event.payload.Revenue)
	assert.Equal(t, "GBP", event.payload.Revenue.Currency)
	assert.Equal(t, "29.99", event.payload.Revenue.Amount)
}

func TestCollector_Properties(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL)

	require.NoError(t, collectFlushClose(t, collector, &analytics_dto.Event{
		Hostname:   "example.com",
		URL:        "/pricing",
		UserAgent:  "Bot",
		Properties: map[string]string{"plan": "pro", "source": "organic"},
		Type:       analytics_dto.EventPageView,
		Timestamp:  time.Now(),
	}))

	props := requireSingleEvent(t, recorder).payload.Props
	assert.Equal(t, "pro", props["plan"])
	assert.Equal(t, "organic", props["source"])
}

func TestCollector_PropsLimitedTo30(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL)

	props := make(map[string]string, 35)
	for i := range 35 {
		props[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("val_%d", i)
	}

	require.NoError(t, collectFlushClose(t, collector, &analytics_dto.Event{
		Hostname:   "example.com",
		URL:        "/test",
		UserAgent:  "Bot",
		Properties: props,
		Type:       analytics_dto.EventPageView,
		Timestamp:  time.Now(),
	}))

	assert.LessOrEqual(t, len(requireSingleEvent(t, recorder).payload.Props), maxProps)
}

func TestCollector_URLTruncation(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL)

	require.NoError(t, collectFlushClose(t, collector, &analytics_dto.Event{
		Hostname:  "example.com",
		URL:       "/" + strings.Repeat("a", maxURLLength+100),
		UserAgent: "Bot",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}))

	assert.LessOrEqual(t, len(requireSingleEvent(t, recorder).payload.URL), maxURLLength)
}

func TestCollector_SelfHostedEndpoint(t *testing.T) {
	server, requestCount := newCountingServer(t, http.StatusAccepted)
	collector := newTestCollector(t, server.URL)

	require.NoError(t, collectFlushClose(t, collector, &analytics_dto.Event{
		Hostname:  "example.com",
		URL:       "/test",
		UserAgent: "Bot",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}))

	assert.Equal(t, int32(1), requestCount.Load())
}

func TestCollector_EmptyDomainReturnsError(t *testing.T) {
	_, err := NewCollector("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain must not be empty")
}

func TestCollector_FlushEmpty(t *testing.T) {
	server, requestCount := newCountingServer(t, http.StatusAccepted)
	collector := newTestCollector(t, server.URL)

	require.NoError(t, collectFlushClose(t, collector))
	assert.Zero(t, requestCount.Load(), "an empty buffer must not POST to Plausible")
}

func TestCollector_FailuresAreReturned(t *testing.T) {
	testCases := []struct {
		serverURL     func(t *testing.T) string
		name          string
		expectedError string
	}{
		{
			name: "error status",
			serverURL: func(t *testing.T) string {
				t.Helper()
				server, _ := newCountingServer(t, http.StatusBadRequest)
				return server.URL
			},
			expectedError: "returned status 400",
		},
		{
			name: "transport failure",
			serverURL: func(t *testing.T) string {
				t.Helper()
				server := httptest.NewServer(http.NotFoundHandler())
				server.Close()
				return server.URL
			},
			expectedError: "posting analytics Plausible event",
		},
		{
			name: "invalid endpoint",
			serverURL: func(t *testing.T) string {
				t.Helper()
				return "http://invalid host"
			},
			expectedError: "creating analytics Plausible request",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			collector := newTestCollector(t, testCase.serverURL(t))

			err := collectFlushClose(t, collector, &analytics_dto.Event{
				Hostname:  "example.com",
				URL:       "/test",
				UserAgent: "Bot",
				Type:      analytics_dto.EventPageView,
				Timestamp: time.Now(),
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.expectedError)
		})
	}
}

func TestCollector_RedirectReplaysRequestBody(t *testing.T) {
	server, recorder := newRecordingServer(t)
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		http.Redirect(w, request, server.URL+eventPath, http.StatusTemporaryRedirect)
	}))
	defer redirector.Close()

	collector := newTestCollector(t, redirector.URL)

	require.NoError(t, collectFlushClose(t, collector,
		&analytics_dto.Event{Hostname: "example.com", URL: "/first", UserAgent: "Bot", Type: analytics_dto.EventPageView},
		&analytics_dto.Event{Hostname: "example.com", URL: "/second", UserAgent: "Bot", Type: analytics_dto.EventPageView},
	))

	received := recorder.events()
	require.Len(t, received, 2)
	assert.Equal(t, "https://example.com/first", received[0].payload.URL)
	assert.Equal(t, "https://example.com/second", received[1].payload.URL)
}

func TestCollector_RequestBodyOutlivesSendEvent(t *testing.T) {
	var requests []*http.Request
	collector := newTestCollector(t, "http://plausible.invalid")
	collector.client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request)
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Body:       http.NoBody,
			Header:     http.Header{},
			Request:    request,
		}, nil
	})

	require.NoError(t, collector.sendEvent(context.Background(), &snapshot{
		payload: eventPayload{Domain: "example.com", Name: "first", URL: "/first"},
	}))
	require.NoError(t, collector.sendEvent(context.Background(), &snapshot{
		payload: eventPayload{Domain: "example.com", Name: "second", URL: "/second"},
	}))
	require.NoError(t, collector.Close(context.Background()))

	require.Len(t, requests, 2)
	for index, expectedName := range []string{"first", "second"} {
		body, err := requests[index].GetBody()
		require.NoError(t, err)
		data, err := io.ReadAll(body)
		require.NoError(t, err)
		require.NoError(t, body.Close())
		assert.Contains(t, string(data), `"name":"`+expectedName+`"`,
			"a request body must not change after sendEvent returns")
	}
}

func TestCollector_Name(t *testing.T) {
	result, err := NewCollector("example.com")
	require.NoError(t, err)
	collector := requireCollector(t, result)
	defer func() { _ = collector.Close(context.Background()) }()

	assert.Equal(t, "plausible", collector.Name())
}

func TestCollector_DoubleClose(t *testing.T) {
	result, err := NewCollector("example.com")
	require.NoError(t, err)
	collector := requireCollector(t, result)

	require.NoError(t, collector.Close(context.Background()))
	require.NoError(t, collector.Close(context.Background()))
}

func TestCollector_MultipleEvents(t *testing.T) {
	server, requestCount := newCountingServer(t, http.StatusAccepted)
	collector := newTestCollector(t, server.URL)

	events := make([]*analytics_dto.Event, 0, 3)
	for range 3 {
		events = append(events, &analytics_dto.Event{
			Hostname:  "example.com",
			URL:       "/page",
			UserAgent: "Bot",
			Type:      analytics_dto.EventPageView,
			Timestamp: time.Now(),
		})
	}
	require.NoError(t, collectFlushClose(t, collector, events...))

	assert.Equal(t, int32(3), requestCount.Load(), "each event is sent as a separate request")
}

func TestCollector_NoClientIP(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL)

	require.NoError(t, collectFlushClose(t, collector, &analytics_dto.Event{
		Hostname:  "example.com",
		URL:       "/test",
		UserAgent: "Bot",
		ClientIP:  "",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}))

	assert.Empty(t, requireSingleEvent(t, recorder).forwardedFor)
}

func TestCollector_FallbackDomain(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL)

	require.NoError(t, collectFlushClose(t, collector, &analytics_dto.Event{
		URL:       "/test",
		UserAgent: "Bot",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}))

	assert.Equal(t, "test.example.com", requireSingleEvent(t, recorder).payload.Domain)
}

func TestResolveEventName(t *testing.T) {
	tests := []struct {
		event    *analytics_dto.Event
		name     string
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
			assert.Equal(t, testCase.expected, resolveEventName(testCase.event))
		})
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		event    *analytics_dto.Event
		name     string
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
			assert.Equal(t, testCase.expected, resolveURL(testCase.event))
		})
	}
}

func TestWithTimeout(t *testing.T) {
	result, err := NewCollector("example.com", WithTimeout(42*time.Second))
	require.NoError(t, err)
	collector := requireCollector(t, result)
	defer func() { _ = collector.Close(context.Background()) }()

	assert.Equal(t, 42*time.Second, collector.client.Timeout)
}

func TestWithRetry(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	collector := newTestCollector(t, server.URL, WithRetry(analytics.RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Millisecond,
		MaxDelay:      5 * time.Millisecond,
		BackoffFactor: 2.0,
		JitterFunc:    func(time.Duration) time.Duration { return 0 },
	}))

	require.NoError(t, collectFlushClose(t, collector, &analytics_dto.Event{
		Hostname:  "example.com",
		URL:       "/retry-test",
		UserAgent: "Bot",
		Type:      analytics_dto.EventPageView,
		Timestamp: time.Now(),
	}))

	assert.GreaterOrEqual(t, attempts.Load(), int32(3), "expected two failures and one success")
}

func TestWithClock(t *testing.T) {
	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	result, err := NewCollector("example.com", withClock(mockClock))
	require.NoError(t, err)
	collector := requireCollector(t, result)
	defer func() { _ = collector.Close(context.Background()) }()

	assert.Same(t, mockClock, collector.clock)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
