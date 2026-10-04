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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

type payloadRecorder struct {
	received []payload
	mu       sync.Mutex
}

func (r *payloadRecorder) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if !assert.NoError(t, err, "reading request body") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p payload
		if !assert.NoError(t, json.Unmarshal(body, &p), "unmarshalling GA4 body") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		r.mu.Lock()
		r.received = append(r.received, p)
		r.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}
}

func (r *payloadRecorder) payloads() []payload {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]payload(nil), r.received...)
}

func newRecordingServer(t *testing.T) (*httptest.Server, *payloadRecorder) {
	t.Helper()
	recorder := &payloadRecorder{}
	server := httptest.NewServer(recorder.handler(t))
	t.Cleanup(server.Close)
	return server, recorder
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
	result, err := NewCollector("G-TEST", "test-secret", allOpts...)
	require.NoError(t, err)
	collector := requireCollector(t, result)
	collector.endpoint = testURL
	collector.Start(context.Background())
	return collector
}

func collectFlushClose(t *testing.T, collector *Collector, events ...*analytics_dto.Event) {
	t.Helper()
	ctx := context.Background()
	for _, event := range events {
		require.NoError(t, collector.Collect(ctx, event))
	}
	require.NoError(t, collector.Flush(ctx))
	require.NoError(t, collector.Close(ctx))
}

func requireSingleEvent(t *testing.T, recorder *payloadRecorder) (payload, ga4Event) {
	t.Helper()
	received := recorder.payloads()
	require.Len(t, received, 1)
	require.Len(t, received[0].Events, 1)
	return received[0], received[0].Events[0]
}

func TestCollector_BatchFlush(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL, WithBatchSize(3))

	events := make([]*analytics_dto.Event, 0, 3)
	for range 3 {
		events = append(events, &analytics_dto.Event{
			Path:       "/page",
			Method:     http.MethodGet,
			StatusCode: 200,
			Type:       analytics_dto.EventPageView,
			Timestamp:  time.Now(),
			ClientIP:   "1.2.3.4",
			UserAgent:  "TestBot",
		})
	}
	collectFlushClose(t, collector, events...)

	received := recorder.payloads()
	require.Len(t, received, 1)
	require.Len(t, received[0].Events, 3)
	assert.Equal(t, "page_view", received[0].Events[0].Name)
}

func TestCollector_PageViewMapping(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL, WithBatchSize(1))

	collectFlushClose(t, collector, &analytics_dto.Event{
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
	})

	_, event := requireSingleEvent(t, recorder)
	assert.Equal(t, "page_view", event.Name)
	assert.Equal(t, "https://example.com/products?cat=shoes", event.Params["page_location"])
	assert.Equal(t, "https://google.com", event.Params["page_referrer"])
	assert.Equal(t, "en-GB", event.Params["language"])
}

func TestCollector_EventNameMapping(t *testing.T) {
	testCases := []struct {
		event          *analytics_dto.Event
		expectedParams map[string]any
		name           string
		expectedName   string
	}{
		{
			name: "custom event name is used",
			event: &analytics_dto.Event{
				EventName:  "purchase",
				Path:       "/checkout",
				StatusCode: 200,
				Type:       analytics_dto.EventCustom,
				Timestamp:  time.Now(),
				ClientIP:   "10.0.0.1",
			},
			expectedName: "purchase",
		},
		{
			name: "action name becomes the event name and a parameter",
			event: &analytics_dto.Event{
				ActionName: "cart.Purchase",
				Path:       "/api/action",
				StatusCode: 200,
				Type:       analytics_dto.EventAction,
				Timestamp:  time.Now(),
				ClientIP:   "10.0.0.1",
			},
			expectedName:   "cart.Purchase",
			expectedParams: map[string]any{"action_name": "cart.Purchase"},
		},
		{
			name: "properties are merged into parameters",
			event: &analytics_dto.Event{
				Path:       "/signup",
				StatusCode: 200,
				Type:       analytics_dto.EventCustom,
				Properties: map[string]string{"plan": "pro", "source": "organic"},
				Timestamp:  time.Now(),
				ClientIP:   "10.0.0.1",
			},
			expectedParams: map[string]any{"plan": "pro", "source": "organic"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server, recorder := newRecordingServer(t)
			collector := newTestCollector(t, server.URL, WithBatchSize(1))

			collectFlushClose(t, collector, testCase.event)

			_, event := requireSingleEvent(t, recorder)
			if testCase.expectedName != "" {
				assert.Equal(t, testCase.expectedName, event.Name)
			}
			for key, value := range testCase.expectedParams {
				assert.Equal(t, value, event.Params[key], "parameter %q", key)
			}
		})
	}
}

func TestCollector_RevenueMapping(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL, WithBatchSize(1))

	collectFlushClose(t, collector, &analytics_dto.Event{
		EventName:  "purchase",
		Path:       "/checkout",
		StatusCode: 200,
		Type:       analytics_dto.EventCustom,
		Revenue:    new(maths.NewMoneyFromString("49.99", "GBP")),
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	})

	_, event := requireSingleEvent(t, recorder)
	assert.Equal(t, "GBP", event.Params["currency"])
	value, ok := event.Params["value"].(float64)
	require.Truef(t, ok, "value is not float64: %T", event.Params["value"])
	assert.InDelta(t, 49.99, value, 1e-9)
}

func TestCollector_ClientIDFromIPAndUA(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL, WithBatchSize(1))

	collectFlushClose(t, collector, &analytics_dto.Event{
		Path:       "/test",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "192.168.1.1",
		UserAgent:  "TestBrowser/1.0",
	})

	expectedHash := sha256.New()
	expectedHash.Write([]byte("192.168.1.1"))
	expectedHash.Write([]byte("|"))
	expectedHash.Write([]byte("TestBrowser/1.0"))

	received, _ := requireSingleEvent(t, recorder)
	assert.Equal(t, hex.EncodeToString(expectedHash.Sum(nil)), received.ClientID)
}

func TestCollector_CustomClientIDFunc(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL,
		WithBatchSize(1),
		WithClientIDFunc(func(_, _ string) string { return "custom-client-123" }),
	)

	collectFlushClose(t, collector, &analytics_dto.Event{
		Path:       "/test",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	})

	received, _ := requireSingleEvent(t, recorder)
	assert.Equal(t, "custom-client-123", received.ClientID)
}

func TestCollector_UserIDMapping(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL, WithBatchSize(1))

	collectFlushClose(t, collector, &analytics_dto.Event{
		Path:       "/account",
		StatusCode: 200,
		UserID:     "user-alex-demo",
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	})

	received, _ := requireSingleEvent(t, recorder)
	assert.Equal(t, "user-alex-demo", received.UserID)
}

func TestCollector_TimestampMicros(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL, WithBatchSize(1))

	eventTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	collectFlushClose(t, collector, &analytics_dto.Event{
		Path:       "/test",
		StatusCode: 200,
		Timestamp:  eventTime,
		ClientIP:   "10.0.0.1",
	})

	received, _ := requireSingleEvent(t, recorder)
	assert.Equal(t, eventTime.UnixMicro(), received.TimestampMicros)
}

func TestCollector_DebugEndpoint(t *testing.T) {
	result, err := NewCollector("G-TEST", "secret", WithDebug(true))
	require.NoError(t, err)
	collector := requireCollector(t, result)
	assert.Contains(t, collector.endpoint, "/debug/mp/collect")
	require.NoError(t, collector.Close(context.Background()))

	resultProduction, err := NewCollector("G-TEST", "secret")
	require.NoError(t, err)
	collectorProduction := requireCollector(t, resultProduction)
	assert.Contains(t, collectorProduction.endpoint, "/mp/collect")
	assert.NotContains(t, collectorProduction.endpoint, "/debug/")
	require.NoError(t, collectorProduction.Close(context.Background()))
}

func TestCollector_FlushEmptyBuffer(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := newTestCollector(t, server.URL)

	require.NoError(t, collector.Flush(context.Background()))
	require.NoError(t, collector.Close(context.Background()))
	assert.Zero(t, requestCount.Load(), "an empty buffer must not POST to GA4")
}

func TestCollector_Name(t *testing.T) {
	result, err := NewCollector("G-TEST", "secret")
	require.NoError(t, err)
	collector := requireCollector(t, result)
	defer func() { _ = collector.Close(context.Background()) }()

	assert.Equal(t, "ga4", collector.Name())
}

func TestCollector_DoubleClose(t *testing.T) {
	result, err := NewCollector("G-TEST", "secret")
	require.NoError(t, err)
	collector := requireCollector(t, result)

	require.NoError(t, collector.Close(context.Background()))
	require.NoError(t, collector.Close(context.Background()))
}

func TestCollector_ErrorStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	collector := newTestCollector(t, server.URL, WithBatchSize(1))

	require.NoError(t, collector.Collect(context.Background(), &analytics_dto.Event{
		Path:       "/error",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}))
	err := collector.Flush(context.Background())
	require.NoError(t, collector.Close(context.Background()))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestCollector_TransportFailureIsReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	serverURL := server.URL
	server.Close()

	collector := newTestCollector(t, serverURL, WithBatchSize(1))

	require.NoError(t, collector.Collect(context.Background(), &analytics_dto.Event{
		Path:      "/unreachable",
		Timestamp: time.Now(),
		ClientIP:  "10.0.0.1",
	}))
	err := collector.Flush(context.Background())
	require.NoError(t, collector.Close(context.Background()))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "posting analytics GA4 batch")
}

func TestCollector_RedirectReplaysRequestBody(t *testing.T) {
	server, recorder := newRecordingServer(t)
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		http.Redirect(w, request, server.URL, http.StatusTemporaryRedirect)
	}))
	defer redirector.Close()

	collector := newTestCollector(t, redirector.URL, WithBatchSize(1))

	collectFlushClose(t, collector, &analytics_dto.Event{
		EventName: "redirected",
		Path:      "/redirect",
		Timestamp: time.Now(),
		ClientIP:  "10.0.0.1",
	})

	_, event := requireSingleEvent(t, recorder)
	assert.Equal(t, "redirected", event.Name)
}

func TestCollector_EncodeChunkReturnsIndependentBodies(t *testing.T) {
	collector := &Collector{}
	timestamp := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)

	first, err := collector.encodeChunk("client-a", "", []eventSnapshot{{name: "first", timestamp: timestamp}})
	require.NoError(t, err)
	firstCopy := string(first)

	second, err := collector.encodeChunk("client-b", "", []eventSnapshot{{name: "second", timestamp: timestamp}})
	require.NoError(t, err)

	assert.Equal(t, firstCopy, string(first), "encoding a second chunk must not overwrite the first body")
	assert.NotEqual(t, string(first), string(second))
}

func TestCollector_TimerFlush(t *testing.T) {
	server, recorder := newRecordingServer(t)

	mockClock := clock.NewMockClock(time.Now())
	collector := newTestCollector(t, server.URL,
		WithBatchSize(100),
		WithFlushInterval(5*time.Second),
		withClock(mockClock),
	)

	require.NoError(t, collector.Collect(context.Background(), &analytics_dto.Event{
		Path:       "/timer",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	}))

	mockClock.Advance(6 * time.Second)

	require.NoError(t, collector.Flush(context.Background()))
	assert.NotEmpty(t, recorder.payloads(), "expected timer-based flush to send events")
	require.NoError(t, collector.Close(context.Background()))
}

func TestCollector_MaxBatchClamped(t *testing.T) {
	result, err := NewCollector("G-TEST", "secret", WithBatchSize(50))
	require.NoError(t, err)
	collector := requireCollector(t, result)
	defer func() { _ = collector.Close(context.Background()) }()

	assert.Equal(t, maxEventsPerRequest, collector.batchSize)
}

func TestCollector_ClientIDPartitioning(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL,
		WithBatchSize(10),
		WithFlushInterval(1*time.Hour),
	)

	collectFlushClose(t, collector,
		&analytics_dto.Event{
			Path: "/page1", StatusCode: 200, Timestamp: time.Now(),
			ClientIP: "1.1.1.1", UserAgent: "Bot",
			Type: analytics_dto.EventPageView,
		},
		&analytics_dto.Event{
			Path: "/page2", StatusCode: 200, Timestamp: time.Now(),
			ClientIP: "2.2.2.2", UserAgent: "Bot",
			Type: analytics_dto.EventPageView,
		},
	)

	received := recorder.payloads()
	require.Len(t, received, 2, "expected one payload per client_id")
	assert.NotEqual(t, received[0].ClientID, received[1].ClientID)
}

func TestCollector_AnonymousClientID(t *testing.T) {
	assert.Equal(t, "anonymous", defaultClientID("", ""))
}

func TestCollector_FlushOnShutdown(t *testing.T) {
	server, recorder := newRecordingServer(t)
	collector := newTestCollector(t, server.URL,
		WithBatchSize(100),
		WithFlushInterval(1*time.Hour),
	)

	collectFlushClose(t, collector, &analytics_dto.Event{
		Path:       "/pending",
		StatusCode: 200,
		Timestamp:  time.Now(),
		ClientIP:   "10.0.0.1",
	})

	assert.Len(t, recorder.payloads(), 1)
}

func TestCollector_EndpointQueryParams(t *testing.T) {
	result, err := NewCollector("G-ABCDEF", "my-secret-key")
	require.NoError(t, err)
	collector := requireCollector(t, result)
	defer func() { _ = collector.Close(context.Background()) }()

	assert.Contains(t, collector.endpoint, "measurement_id=G-ABCDEF")
	assert.Contains(t, collector.endpoint, "api_secret=my-secret-key")
}

func TestNewCollector_RejectsMissingCredentials(t *testing.T) {
	testCases := []struct {
		name          string
		measurementID string
		apiSecret     string
		expectedError string
	}{
		{name: "empty measurement ID", measurementID: "", apiSecret: "secret", expectedError: "measurementID"},
		{name: "empty API secret", measurementID: "G-TEST", apiSecret: "", expectedError: "apiSecret"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewCollector(testCase.measurementID, testCase.apiSecret)
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.expectedError)
		})
	}
}
