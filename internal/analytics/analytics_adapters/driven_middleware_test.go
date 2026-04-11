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
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/maths"
)

type testCollector struct {
	events []analytics_dto.Event
	mu     sync.Mutex
}

func (c *testCollector) Collect(_ context.Context, ev *analytics_dto.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, *ev)
	return nil
}

func (c *testCollector) Start(_ context.Context)       {}
func (c *testCollector) Flush(_ context.Context) error { return nil }
func (c *testCollector) Close(_ context.Context) error { return nil }
func (c *testCollector) Name() string                  { return "test" }

func (c *testCollector) collected() []analytics_dto.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]analytics_dto.Event, len(c.events))
	copy(out, c.events)
	return out
}

func newRequestWithPctx(method, path string, pctx *daemon_dto.PikoRequestCtx) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	return req.WithContext(daemon_dto.WithPikoRequestCtx(req.Context(), pctx))
}

func TestMiddleware_FiresPageViewEvent(t *testing.T) {
	tc := &testCollector{}
	svc := analytics_domain.NewService([]analytics_domain.Collector{tc})
	svc.Start(context.Background())

	mw := NewAnalyticsMiddleware(svc, logger_domain.GetLogger("test"))
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	pctx := daemon_dto.AcquirePikoRequestCtx()
	req := newRequestWithPctx(http.MethodGet, "/test-path", pctx)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("Referer", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	svc.Close(context.Background())

	events := tc.collected()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if ev.Path != "/test-path" {
		t.Errorf("Path = %q, want /test-path", ev.Path)
	}
	if ev.Hostname != "example.com" {
		t.Errorf("Hostname = %q, want example.com", ev.Hostname)
	}
	if ev.URL != "/test-path" {
		t.Errorf("URL = %q, want /test-path", ev.URL)
	}
	if ev.Method != http.MethodGet {
		t.Errorf("Method = %q, want GET", ev.Method)
	}
	if ev.UserAgent != "TestAgent/1.0" {
		t.Errorf("UserAgent = %q, want TestAgent/1.0", ev.UserAgent)
	}
	if ev.Referrer != "https://example.com" {
		t.Errorf("Referrer = %q, want https://example.com", ev.Referrer)
	}
	if ev.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", ev.StatusCode)
	}
	if ev.Type != analytics_dto.EventPageView {
		t.Errorf("Type = %v, want EventPageView", ev.Type)
	}
	if ev.Duration <= 0 {
		t.Error("expected Duration > 0")
	}

	daemon_dto.ReleasePikoRequestCtx(pctx)
}

func TestMiddleware_EnrichesFromPikoRequestCtx(t *testing.T) {
	tc := &testCollector{}
	svc := analytics_domain.NewService([]analytics_domain.Collector{tc})
	svc.Start(context.Background())

	mw := NewAnalyticsMiddleware(svc, logger_domain.GetLogger("test"))
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	pctx := daemon_dto.AcquirePikoRequestCtx()
	pctx.ClientIP = "203.0.113.50"
	pctx.Locale = "de"
	pctx.MatchedPattern = "/users/{id}"
	pctx.CachedAuth = &stubAuth{
		authenticated: true,
		userID:        "user-42",
	}

	req := newRequestWithPctx(http.MethodGet, "/users/42", pctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	svc.Close(context.Background())

	events := tc.collected()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if ev.ClientIP != "203.0.113.50" {
		t.Errorf("ClientIP = %q, want 203.0.113.50", ev.ClientIP)
	}
	if ev.Locale != "de" {
		t.Errorf("Locale = %q, want de", ev.Locale)
	}
	if ev.MatchedPattern != "/users/{id}" {
		t.Errorf("MatchedPattern = %q, want /users/{id}", ev.MatchedPattern)
	}
	if ev.UserID != "user-42" {
		t.Errorf("UserID = %q, want user-42", ev.UserID)
	}

	daemon_dto.ReleasePikoRequestCtx(pctx)
}

func TestMiddleware_CapturesNon200StatusCode(t *testing.T) {
	tc := &testCollector{}
	svc := analytics_domain.NewService([]analytics_domain.Collector{tc})
	svc.Start(context.Background())

	mw := NewAnalyticsMiddleware(svc, logger_domain.GetLogger("test"))
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	pctx := daemon_dto.AcquirePikoRequestCtx()
	req := newRequestWithPctx(http.MethodGet, "/missing", pctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	svc.Close(context.Background())

	events := tc.collected()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", events[0].StatusCode)
	}

	daemon_dto.ReleasePikoRequestCtx(pctx)
}

func TestMiddleware_UnauthenticatedUserID(t *testing.T) {
	tc := &testCollector{}
	svc := analytics_domain.NewService([]analytics_domain.Collector{tc})
	svc.Start(context.Background())

	mw := NewAnalyticsMiddleware(svc, logger_domain.GetLogger("test"))
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	pctx := daemon_dto.AcquirePikoRequestCtx()
	pctx.ClientIP = "10.0.0.1"

	req := newRequestWithPctx(http.MethodGet, "/public", pctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	svc.Close(context.Background())

	events := tc.collected()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].UserID != "" {
		t.Errorf("UserID = %q, want empty for unauthenticated", events[0].UserID)
	}

	daemon_dto.ReleasePikoRequestCtx(pctx)
}

func TestMiddleware_DefaultStatusCode(t *testing.T) {
	tc := &testCollector{}
	svc := analytics_domain.NewService([]analytics_domain.Collector{tc})
	svc.Start(context.Background())

	mw := NewAnalyticsMiddleware(svc, logger_domain.GetLogger("test"))
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("hello"))
	}))

	pctx := daemon_dto.AcquirePikoRequestCtx()
	req := newRequestWithPctx(http.MethodGet, "/implicit-200", pctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	svc.Close(context.Background())

	events := tc.collected()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200 (implicit)", events[0].StatusCode)
	}

	daemon_dto.ReleasePikoRequestCtx(pctx)
}

func TestMiddleware_NilPikoRequestCtx(t *testing.T) {
	tc := &testCollector{}
	svc := analytics_domain.NewService([]analytics_domain.Collector{tc})
	svc.Start(context.Background())

	mw := NewAnalyticsMiddleware(svc, logger_domain.GetLogger("test"))
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/no-ctx", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	svc.Close(context.Background())

	events := tc.collected()
	if len(events) != 0 {
		t.Errorf("expected 0 events without PikoRequestCtx, got %d", len(events))
	}
}

func TestMiddleware_StashedRevenuePropertiesAndEventName(t *testing.T) {
	tc := &testCollector{}
	svc := analytics_domain.NewService([]analytics_domain.Collector{tc})
	svc.Start(context.Background())

	mw := NewAnalyticsMiddleware(svc, logger_domain.GetLogger("test"))
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pctx := daemon_dto.PikoRequestCtxFromContext(r.Context())
		revenue := maths.NewMoneyFromString("49.99", "GBP")
		pctx.AnalyticsRevenue = &revenue
		pctx.AnalyticsProperties = map[string]string{"plan": "pro", "source": "email"}
		pctx.AnalyticsEventName = "purchase"
		w.WriteHeader(http.StatusOK)
	}))

	pctx := daemon_dto.AcquirePikoRequestCtx()
	req := newRequestWithPctx(http.MethodPost, "/checkout", pctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	svc.Close(context.Background())

	events := tc.collected()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if ev.Type != analytics_dto.EventCustom {
		t.Errorf("Type = %v, want EventCustom", ev.Type)
	}
	if ev.EventName != "purchase" {
		t.Errorf("EventName = %q, want purchase", ev.EventName)
	}
	if ev.Revenue == nil {
		t.Fatal("Revenue is nil, want non-nil")
	}
	revenueNumber := ev.Revenue.MustNumber()
	if revenueNumber != "49.99" {
		t.Errorf("Revenue amount = %q, want 49.99", revenueNumber)
	}
	currencyCode, err := ev.Revenue.CurrencyCode()
	if err != nil {
		t.Fatalf("Revenue.CurrencyCode() error: %v", err)
	}
	if currencyCode != "GBP" {
		t.Errorf("Revenue currency = %q, want GBP", currencyCode)
	}
	if ev.Properties["plan"] != "pro" {
		t.Errorf("Properties[plan] = %q, want pro", ev.Properties["plan"])
	}
	if ev.Properties["source"] != "email" {
		t.Errorf("Properties[source] = %q, want email", ev.Properties["source"])
	}

	daemon_dto.ReleasePikoRequestCtx(pctx)
}

func TestMiddleware_EventNameChangesTypeToCustom(t *testing.T) {
	tc := &testCollector{}
	svc := analytics_domain.NewService([]analytics_domain.Collector{tc})
	svc.Start(context.Background())

	mw := NewAnalyticsMiddleware(svc, logger_domain.GetLogger("test"))
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pctx := daemon_dto.PikoRequestCtxFromContext(r.Context())
		pctx.AnalyticsEventName = "signup"
		w.WriteHeader(http.StatusOK)
	}))

	pctx := daemon_dto.AcquirePikoRequestCtx()
	req := newRequestWithPctx(http.MethodGet, "/register", pctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	svc.Close(context.Background())

	events := tc.collected()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if ev.Type != analytics_dto.EventCustom {
		t.Errorf("Type = %v, want EventCustom when AnalyticsEventName is set", ev.Type)
	}
	if ev.EventName != "signup" {
		t.Errorf("EventName = %q, want signup", ev.EventName)
	}

	daemon_dto.ReleasePikoRequestCtx(pctx)
}

func TestMiddleware_NoEventNameKeepsPageView(t *testing.T) {
	tc := &testCollector{}
	svc := analytics_domain.NewService([]analytics_domain.Collector{tc})
	svc.Start(context.Background())

	mw := NewAnalyticsMiddleware(svc, logger_domain.GetLogger("test"))
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	pctx := daemon_dto.AcquirePikoRequestCtx()
	req := newRequestWithPctx(http.MethodGet, "/page", pctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	svc.Close(context.Background())

	events := tc.collected()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != analytics_dto.EventPageView {
		t.Errorf("Type = %v, want EventPageView when no AnalyticsEventName set", events[0].Type)
	}
	if events[0].EventName != "" {
		t.Errorf("EventName = %q, want empty", events[0].EventName)
	}

	daemon_dto.ReleasePikoRequestCtx(pctx)
}

type stubAuth struct {
	userID        string
	authenticated bool
}

func (a *stubAuth) IsAuthenticated() bool { return a.authenticated }
func (a *stubAuth) UserID() string        { return a.userID }
func (a *stubAuth) Get(_ string) any      { return nil }
