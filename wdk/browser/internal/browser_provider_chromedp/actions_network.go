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
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

// NetworkRequest holds the details of a captured HTTP request and its response.
type NetworkRequest struct {
	// Timestamp is when the request was created.
	Timestamp time.Time

	// Headers contains HTTP header key-value pairs for the request.
	Headers map[string]string

	// URL is the full request URL.
	URL string

	// Method is the HTTP method (GET, POST, etc.); empty matches any method.
	Method string

	// PostData is the request body content for POST requests.
	PostData string

	// RequestID is the unique identifier for this network request.
	RequestID string

	// Response contains the raw response data from the network request.
	Response string

	// StatusCode is the HTTP status code returned by the server.
	StatusCode int
}

// RequestMatcher defines criteria for matching network requests.
type RequestMatcher struct {
	// URLContains is a substring that must appear in the request URL to match.
	URLContains string

	// URLPattern is the pattern to match against request URLs.
	URLPattern string

	// Method specifies the HTTP method to match; empty matches any method.
	Method string
}

// MockResponse defines a mock response for intercepted requests.
type MockResponse struct {
	// Headers contains HTTP header key-value pairs for the mock response.
	Headers map[string]string

	// Body is the response body content to return.
	Body string

	// StatusCode is the HTTP status code for the mocked response.
	StatusCode int
}

// NetworkTracker tracks network requests for a page.
type NetworkTracker struct {
	// requests stores the network requests captured during tracking.
	requests []NetworkRequest

	// mu guards concurrent access to the requests slice.
	mu sync.RWMutex
}

// NewNetworkTracker creates a new network tracker.
// Call StartTracking to begin capturing requests.
//
// Returns *NetworkTracker which is ready to track network requests.
func NewNetworkTracker() *NetworkTracker {
	return &NetworkTracker{
		requests: make([]NetworkRequest, 0),
		mu:       sync.RWMutex{},
	}
}

// StartTracking begins capturing network requests.
//
// Takes ctx (*ActionContext) which provides the browser action context.
//
// Returns error when enabling the network domain or setting up interceptors
// fails.
//
// Safe for concurrent use. Protects internal state with a mutex.
func (nt *NetworkTracker) StartTracking(ctx *ActionContext) error {
	nt.mu.Lock()
	defer nt.mu.Unlock()

	nt.requests = make([]NetworkRequest, 0)

	if err := nt.enableNetworkDomain(ctx); err != nil {
		return err
	}

	return nt.setupNetworkInterceptors(ctx)
}

// StopTracking stops capturing network requests.
//
// Takes ctx (*ActionContext) which provides the browser context to operate on.
//
// Returns error when the network tracking cannot be disabled.
func (*NetworkTracker) StopTracking(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return network.Disable().Do(ctx2)
	}))
}

// GetRequests returns all captured network requests.
//
// Takes ctx (*ActionContext) which provides the action context for retrieval.
//
// Returns []NetworkRequest which contains all captured network requests.
// Returns error when the requests cannot be retrieved.
func (*NetworkTracker) GetRequests(ctx *ActionContext) ([]NetworkRequest, error) {
	return getTrackedRequests(ctx)
}

// ClearRequests clears all captured network requests.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
//
// Returns error when the JavaScript execution fails.
func (*NetworkTracker) ClearRequests(ctx *ActionContext) error {
	js := scripts.MustGet("network_requests_clear.js")
	return chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
}

// enableNetworkDomain enables the CDP network domain.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
//
// Returns error when enabling network tracking fails.
func (*NetworkTracker) enableNetworkDomain(ctx *ActionContext) error {
	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return network.Enable().Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("enabling network tracking: %w", err)
	}
	return nil
}

// setupNetworkInterceptors sets up JavaScript interceptors for fetch and XHR.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
//
// Returns error when the JavaScript injection fails.
func (*NetworkTracker) setupNetworkInterceptors(ctx *ActionContext) error {
	return chromedp.Run(ctx.Ctx, chromedp.Evaluate(scripts.MustGet("network_tracking.js"), nil))
}

// RequestInterceptor handles request interception for mocking responses.
type RequestInterceptor struct {
	// mocks maps URL patterns to their mock responses.
	mocks map[string]MockResponse

	// patterns holds URL patterns in the order they were added.
	patterns []string

	// mu guards access to mocks and patterns.
	mu sync.RWMutex

	// enabled indicates whether the request interceptor is active.
	enabled bool
}

// NewRequestInterceptor creates a new request interceptor.
//
// Returns *RequestInterceptor which is ready for use but disabled by default.
func NewRequestInterceptor() *RequestInterceptor {
	return &RequestInterceptor{
		mocks:    make(map[string]MockResponse),
		patterns: make([]string, 0),
		mu:       sync.RWMutex{},
		enabled:  false,
	}
}

// AddMock adds a mock response for requests matching the URL pattern.
//
// Takes urlPattern (string) which specifies the URL pattern to match.
// Takes response (MockResponse) which provides the mock response to return.
//
// Safe for concurrent use.
func (ri *RequestInterceptor) AddMock(urlPattern string, response MockResponse) {
	ri.mu.Lock()
	defer ri.mu.Unlock()
	ri.mocks[urlPattern] = response
	ri.patterns = append(ri.patterns, urlPattern)
}

// RemoveMock removes a mock response for the given URL pattern.
//
// Takes urlPattern (string) which is the pattern to remove from the mock
// registry.
//
// Safe for concurrent use.
func (ri *RequestInterceptor) RemoveMock(urlPattern string) {
	ri.mu.Lock()
	defer ri.mu.Unlock()
	delete(ri.mocks, urlPattern)

	newPatterns := make([]string, 0, len(ri.patterns))
	for _, p := range ri.patterns {
		if p != urlPattern {
			newPatterns = append(newPatterns, p)
		}
	}
	ri.patterns = newPatterns
}

// ClearMocks removes all mock responses.
//
// Safe for concurrent use.
func (ri *RequestInterceptor) ClearMocks() {
	ri.mu.Lock()
	defer ri.mu.Unlock()
	ri.mocks = make(map[string]MockResponse)
	ri.patterns = make([]string, 0)
}

// Enable enables request interception for the configured URL patterns.
//
// Takes ctx (*ActionContext) which provides the browser context for enabling
// interception.
//
// Returns error when the Chrome DevTools fetch.Enable command fails.
//
// Safe for concurrent use. Uses a mutex to protect the enabled state.
func (ri *RequestInterceptor) Enable(ctx *ActionContext) error {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	if ri.enabled {
		return nil
	}

	patterns := make([]*fetch.RequestPattern, 0, len(ri.patterns))
	for _, p := range ri.patterns {
		patterns = append(patterns, &fetch.RequestPattern{
			URLPattern:   p,
			ResourceType: "",
			RequestStage: fetch.RequestStageRequest,
		})
	}

	if len(patterns) == 0 {
		patterns = append(patterns, &fetch.RequestPattern{
			URLPattern:   "*",
			ResourceType: "",
			RequestStage: fetch.RequestStageRequest,
		})
	}

	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return fetch.Enable().WithPatterns(patterns).Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("enabling request interception: %w", err)
	}

	ri.enabled = true
	return nil
}

// Disable disables request interception.
//
// Takes ctx (*ActionContext) which provides the browser context for the action.
//
// Returns error when the fetch disable command fails.
//
// Safe for concurrent use; acquires a mutex lock before modifying state.
func (ri *RequestInterceptor) Disable(ctx *ActionContext) error {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	if !ri.enabled {
		return nil
	}

	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return fetch.Disable().Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("disabling request interception: %w", err)
	}

	ri.enabled = false
	return nil
}

// WaitForNetworkIdle waits until no network requests have been made for the
// specified duration.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes idleDuration (time.Duration) which specifies how long the network must
// be idle before returning.
// Takes timeout (time.Duration) which sets the maximum time to wait for idle.
//
// Returns error when the JavaScript execution fails or the context is invalid.
func WaitForNetworkIdle(ctx *ActionContext, idleDuration time.Duration, timeout time.Duration) error {
	js := scripts.MustExecute("wait_network_idle.js.tmpl", map[string]any{
		"IdleDuration": idleDuration.Milliseconds(),
		"Timeout":      timeout.Milliseconds(),
	})

	var result bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result, chromedp.EvalAsValue))
	if err != nil {
		return fmt.Errorf("waiting for network idle: %w", err)
	}
	return nil
}

// CheckRequestMade checks if a request matching the criteria was made.
//
// Takes ctx (*ActionContext) which provides the browser context to search.
// Takes matcher (RequestMatcher) which specifies the criteria to match against.
//
// Returns bool which is true if a matching request was found.
// Returns error when the tracked requests cannot be retrieved.
func CheckRequestMade(ctx *ActionContext, matcher RequestMatcher) (bool, error) {
	requests, err := getTrackedRequests(ctx)
	if err != nil {
		return false, err
	}

	for _, request := range requests {
		if matchesRequest(request, matcher) {
			return true, nil
		}
	}
	return false, nil
}

// WaitForRequest waits for a request matching the criteria to be made.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes matcher (RequestMatcher) which specifies the criteria to match.
// Takes timeout (time.Duration) which sets how long to wait before failing.
//
// Returns *NetworkRequest which is the first request matching the criteria.
// Returns error when the timeout is reached before a matching request appears.
func WaitForRequest(ctx *ActionContext, matcher RequestMatcher, timeout time.Duration) (*NetworkRequest, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		requests, err := getTrackedRequests(ctx)
		if err != nil {
			return nil, err
		}

		for _, request := range requests {
			if matchesRequest(request, matcher) {
				return &request, nil
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout waiting for request matching %+v", matcher)
}

// InterceptRequest sets up a simple request interceptor using JavaScript.
// This is easier to use than CDP fetch for basic mocking scenarios.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes urlPattern (string) which specifies the URL pattern to intercept.
// Takes mock (MockResponse) which defines the response to return for matches.
//
// Returns error when the JavaScript execution fails.
func InterceptRequest(ctx *ActionContext, urlPattern string, mock MockResponse) error {
	headersJSON := "{}"
	if len(mock.Headers) > 0 {
		pairs := make([]string, 0, len(mock.Headers))
		for k, v := range mock.Headers {
			pairs = append(pairs, fmt.Sprintf(`%q: %q`, k, v))
		}
		headersJSON = "{" + strings.Join(pairs, ", ") + "}"
	}

	js := scripts.MustExecute("mock_request.js.tmpl", map[string]any{
		"URLPattern":  urlPattern,
		"StatusCode":  mock.StatusCode,
		"Body":        mock.Body,
		"HeadersJSON": headersJSON,
	})

	return chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
}

// RemoveRequestIntercept removes a request intercept for the given URL pattern.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes urlPattern (string) which specifies the URL pattern to stop intercepting.
//
// Returns error when the browser command fails to execute.
func RemoveRequestIntercept(ctx *ActionContext, urlPattern string) error {
	js := scripts.MustExecute("unmock_request.js.tmpl", map[string]any{
		"URLPattern": urlPattern,
	})

	return chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
}

// ClearRequestIntercepts removes all request intercepts.
//
// Takes ctx (*ActionContext) which provides the browser context.
//
// Returns error when the script execution fails.
func ClearRequestIntercepts(ctx *ActionContext) error {
	js := scripts.MustGet("request_mocks_clear.js")
	return chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
}

// getString safely extracts a string from a map.
//
// Takes m (map[string]any) which is the map to extract from.
// Takes key (string) which is the key to look up.
//
// Returns string which is the value if found, or an empty string otherwise.
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getTrackedRequests retrieves tracked requests from the page.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
//
// Returns []NetworkRequest which contains the network requests tracked on the
// page.
// Returns error when the JavaScript evaluation fails.
func getTrackedRequests(ctx *ActionContext) ([]NetworkRequest, error) {
	js := `window.__networkRequests || []`

	var result []any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting tracked requests: %w", err)
	}

	requests := make([]NetworkRequest, 0, len(result))
	for _, r := range result {
		if m, ok := r.(map[string]any); ok {
			requests = append(requests, parseTrackedRequestEntry(m))
		}
	}
	return requests, nil
}

// parseTrackedRequestEntry converts a JavaScript object (map) into a
// NetworkRequest.
//
// Takes m (map[string]any) which holds the raw key-value pairs from the
// browser-side tracker.
//
// Returns NetworkRequest which contains the parsed request data.
func parseTrackedRequestEntry(m map[string]any) NetworkRequest {
	var statusCode int
	if status, ok := m["status"].(float64); ok {
		statusCode = int(status)
	}
	return NetworkRequest{
		Timestamp:  time.Time{},
		Headers:    nil,
		URL:        getString(m, "url"),
		Method:     getString(m, "method"),
		PostData:   "",
		RequestID:  "",
		Response:   "",
		StatusCode: statusCode,
	}
}

// matchesRequest checks if a request matches the given criteria.
//
// Takes request (NetworkRequest) which is the request to check.
// Takes matcher (RequestMatcher) which specifies the match criteria.
//
// Returns bool which is true if the request matches all specified criteria.
func matchesRequest(request NetworkRequest, matcher RequestMatcher) bool {
	if matcher.Method != "" && request.Method != matcher.Method {
		return false
	}
	if matcher.URLContains != "" && !strings.Contains(request.URL, matcher.URLContains) {
		return false
	}
	return true
}
