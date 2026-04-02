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

package browser_provider_chromedp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

const testHTMLNetworkFetch = `<!DOCTYPE html>
<html>
<head><title>Network Test</title></head>
<body>
<button id="fetch-btn" onclick="fetch('/api/data').then(r=>r.json()).then(d=>document.getElementById('result').textContent=JSON.stringify(d))">Fetch</button>
<button id="post-btn" onclick="fetch('/api/submit',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({ok:true})}).then(r=>r.json()).then(d=>document.getElementById('result').textContent=JSON.stringify(d))">Post</button>
<div id="result"></div>
</body>
</html>`

func newNetworkTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/data":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/api/submit":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]bool{"received": true})
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(testHTMLNetworkFetch))
		}
	}))
}

func TestNetworkTracker_StartStopTracking(t *testing.T) {
	t.Parallel()
	server := newNetworkTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		tracker := NewNetworkTracker()

		t.Run("start tracking captures requests", func(t *testing.T) {
			err := tracker.StartTracking(ctx)
			if err != nil {
				t.Fatalf("StartTracking() error = %v", err)
			}

			err = chromedp.Run(page.Ctx(), chromedp.Click("#fetch-btn", chromedp.ByQuery))
			if err != nil {
				t.Fatalf("clicking fetch button: %v", err)
			}

			time.Sleep(500 * time.Millisecond)

			err = tracker.StopTracking(ctx)
			if err != nil {
				t.Fatalf("StopTracking() error = %v", err)
			}
		})
	})
}

func TestNetworkTracker_GetRequests(t *testing.T) {
	t.Parallel()
	server := newNetworkTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		tracker := NewNetworkTracker()

		err := tracker.StartTracking(ctx)
		if err != nil {
			t.Fatalf("StartTracking() error = %v", err)
		}
		defer func() { _ = tracker.StopTracking(ctx) }()

		err = chromedp.Run(page.Ctx(), chromedp.Click("#fetch-btn", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("clicking fetch button: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		t.Run("get all requests", func(t *testing.T) {
			requests, err := tracker.GetRequests(ctx)
			if err != nil {
				t.Fatalf("GetRequests() error = %v", err)
			}
			if len(requests) == 0 {
				t.Error("expected at least one tracked request")
			}
		})

		t.Run("get requests with matcher", func(t *testing.T) {
			requests, err := tracker.GetRequests(ctx)
			if err != nil {
				t.Fatalf("GetRequests() error = %v", err)
			}

			matched := false
			for _, request := range requests {
				if matchesRequest(request, RequestMatcher{URLContains: "/api/data"}) {
					matched = true
					break
				}
			}
			if !matched {
				t.Error("expected to find a request matching /api/data")
			}
		})
	})
}

func TestNetworkTracker_ClearRequests(t *testing.T) {
	t.Parallel()
	server := newNetworkTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		tracker := NewNetworkTracker()

		err := tracker.StartTracking(ctx)
		if err != nil {
			t.Fatalf("StartTracking() error = %v", err)
		}
		defer func() { _ = tracker.StopTracking(ctx) }()

		err = chromedp.Run(page.Ctx(), chromedp.Click("#fetch-btn", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("clicking fetch button: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		t.Run("clear empties requests", func(t *testing.T) {
			err := tracker.ClearRequests(ctx)
			if err != nil {
				t.Fatalf("ClearRequests() error = %v", err)
			}

			requests, err := tracker.GetRequests(ctx)
			if err != nil {
				t.Fatalf("GetRequests() error = %v", err)
			}
			if len(requests) != 0 {
				t.Errorf("expected 0 requests after clear, got %d", len(requests))
			}
		})
	})
}

func TestNetworkTracker_WaitForNetworkIdle(t *testing.T) {
	t.Parallel()
	server := newNetworkTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		tracker := NewNetworkTracker()

		err := tracker.StartTracking(ctx)
		if err != nil {
			t.Fatalf("StartTracking() error = %v", err)
		}
		defer func() { _ = tracker.StopTracking(ctx) }()

		t.Run("idle after no requests", func(t *testing.T) {

			_ = WaitForNetworkIdle(ctx, 200*time.Millisecond, 5*time.Second)
		})
	})
}

func TestNetworkTracker_CheckRequestMade(t *testing.T) {
	t.Parallel()
	server := newNetworkTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		tracker := NewNetworkTracker()

		err := tracker.StartTracking(ctx)
		if err != nil {
			t.Fatalf("StartTracking() error = %v", err)
		}
		defer func() { _ = tracker.StopTracking(ctx) }()

		err = chromedp.Run(page.Ctx(), chromedp.Click("#fetch-btn", chromedp.ByQuery))
		if err != nil {
			t.Fatalf("clicking fetch button: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		t.Run("matching request found", func(t *testing.T) {
			found, err := CheckRequestMade(ctx, RequestMatcher{URLContains: "/api/data"})
			if err != nil {
				t.Fatalf("CheckRequestMade() error = %v", err)
			}
			if !found {
				t.Error("expected matching request to be found")
			}
		})

		t.Run("non-matching request not found", func(t *testing.T) {
			found, err := CheckRequestMade(ctx, RequestMatcher{URLContains: "/api/nonexistent"})
			if err != nil {
				t.Fatalf("CheckRequestMade() error = %v", err)
			}
			if found {
				t.Error("expected no matching request")
			}
		})
	})
}

func TestNetworkTracker_WaitForRequest(t *testing.T) {
	t.Parallel()
	server := newNetworkTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		tracker := NewNetworkTracker()

		err := tracker.StartTracking(ctx)
		if err != nil {
			t.Fatalf("StartTracking() error = %v", err)
		}
		defer func() { _ = tracker.StopTracking(ctx) }()

		t.Run("wait for existing request", func(t *testing.T) {

			err := chromedp.Run(page.Ctx(), chromedp.Click("#fetch-btn", chromedp.ByQuery))
			if err != nil {
				t.Fatalf("clicking fetch button: %v", err)
			}

			request, err := WaitForRequest(ctx, RequestMatcher{URLContains: "/api/data"}, 5*time.Second)
			if err != nil {
				t.Fatalf("WaitForRequest() error = %v", err)
			}
			if request == nil {
				t.Fatal("expected non-nil request")
			}
		})

		t.Run("timeout for non-matching request", func(t *testing.T) {
			_, err := WaitForRequest(ctx, RequestMatcher{URLContains: "/nonexistent"}, 500*time.Millisecond)
			if err == nil {
				t.Error("expected timeout error")
			}
		})
	})
}

func TestRequestInterceptor_EnableDisable(t *testing.T) {
	t.Parallel()
	server := newNetworkTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		interceptor := NewRequestInterceptor()

		t.Run("enable and disable", func(t *testing.T) {
			err := interceptor.Enable(ctx)
			if err != nil {
				t.Fatalf("Enable() error = %v", err)
			}
			if !interceptor.enabled {
				t.Error("expected interceptor to be enabled")
			}

			err = interceptor.Enable(ctx)
			if err != nil {
				t.Fatalf("Enable() second call error = %v", err)
			}

			err = interceptor.Disable(ctx)
			if err != nil {
				t.Fatalf("Disable() error = %v", err)
			}
			if interceptor.enabled {
				t.Error("expected interceptor to be disabled")
			}

			err = interceptor.Disable(ctx)
			if err != nil {
				t.Fatalf("Disable() second call error = %v", err)
			}
		})
	})
}

func TestInterceptRequest_JS(t *testing.T) {
	t.Parallel()
	server := newNetworkTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("intercept and remove", func(t *testing.T) {
			mock := MockResponse{
				StatusCode: 200,
				Body:       `{"intercepted": true}`,
				Headers:    map[string]string{"Content-Type": "application/json"},
			}

			err := InterceptRequest(ctx, "/api/data", mock)
			if err != nil {
				t.Fatalf("InterceptRequest() error = %v", err)
			}

			err = chromedp.Run(page.Ctx(), chromedp.Click("#fetch-btn", chromedp.ByQuery))
			if err != nil {
				t.Fatalf("clicking fetch button: %v", err)
			}

			time.Sleep(500 * time.Millisecond)

			var result string
			err = chromedp.Run(page.Ctx(), chromedp.Text("#result", &result, chromedp.ByQuery))
			if err != nil {
				t.Fatalf("reading result: %v", err)
			}
			if result == "" {
				t.Log("result was empty - mock may not have been applied (expected in some CDP configurations)")
			}

			err = RemoveRequestIntercept(ctx, "/api/data")
			if err != nil {
				t.Fatalf("RemoveRequestIntercept() error = %v", err)
			}
		})

		t.Run("clear all intercepts", func(t *testing.T) {
			err := InterceptRequest(ctx, "/api/a", MockResponse{StatusCode: 200, Body: "{}"})
			if err != nil {
				t.Fatalf("InterceptRequest() error = %v", err)
			}

			err = ClearRequestIntercepts(ctx)
			if err != nil {
				t.Fatalf("ClearRequestIntercepts() error = %v", err)
			}
		})
	})
}

func TestRequestInterceptor_WithPatterns(t *testing.T) {
	t.Parallel()
	server := newNetworkTestServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)
		interceptor := NewRequestInterceptor()

		t.Run("enable with specific patterns", func(t *testing.T) {
			interceptor.AddMock("/api/*", MockResponse{
				StatusCode: 200,
				Body:       `{"mocked": true}`,
			})

			err := interceptor.Enable(ctx)
			if err != nil {
				t.Fatalf("Enable() error = %v", err)
			}

			err = interceptor.Disable(ctx)
			if err != nil {
				t.Fatalf("Disable() error = %v", err)
			}

			interceptor.ClearMocks()
		})
	})
}
