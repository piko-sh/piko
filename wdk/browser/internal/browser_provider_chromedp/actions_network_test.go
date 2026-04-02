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
	"testing"
)

func TestGetString(t *testing.T) {
	testCases := []struct {
		name     string
		m        map[string]any
		key      string
		expected string
	}{
		{
			name:     "key exists with string value",
			m:        map[string]any{"url": "https://example.com"},
			key:      "url",
			expected: "https://example.com",
		},
		{
			name:     "key exists with non-string value",
			m:        map[string]any{"status": 200},
			key:      "status",
			expected: "",
		},
		{
			name:     "key missing",
			m:        map[string]any{"other": "value"},
			key:      "url",
			expected: "",
		},
		{
			name:     "nil map",
			m:        nil,
			key:      "url",
			expected: "",
		},
		{
			name:     "empty map",
			m:        map[string]any{},
			key:      "url",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getString(tc.m, tc.key)
			if result != tc.expected {
				t.Errorf("getString(%v, %q) = %q, expected %q", tc.m, tc.key, result, tc.expected)
			}
		})
	}
}

func TestMatchesRequest(t *testing.T) {
	testCases := []struct {
		matcher  RequestMatcher
		name     string
		request  NetworkRequest
		expected bool
	}{
		{
			name:     "empty matcher matches everything",
			request:  NetworkRequest{URL: "https://example.com/api", Method: "GET"},
			matcher:  RequestMatcher{},
			expected: true,
		},
		{
			name:     "method match",
			request:  NetworkRequest{URL: "https://example.com/api", Method: "POST"},
			matcher:  RequestMatcher{Method: "POST"},
			expected: true,
		},
		{
			name:     "method mismatch",
			request:  NetworkRequest{URL: "https://example.com/api", Method: "GET"},
			matcher:  RequestMatcher{Method: "POST"},
			expected: false,
		},
		{
			name:     "URLContains match",
			request:  NetworkRequest{URL: "https://example.com/api/users", Method: "GET"},
			matcher:  RequestMatcher{URLContains: "/api/users"},
			expected: true,
		},
		{
			name:     "URLContains mismatch",
			request:  NetworkRequest{URL: "https://example.com/api/posts", Method: "GET"},
			matcher:  RequestMatcher{URLContains: "/api/users"},
			expected: false,
		},
		{
			name:     "both method and URL match",
			request:  NetworkRequest{URL: "https://example.com/api/users", Method: "POST"},
			matcher:  RequestMatcher{Method: "POST", URLContains: "/api/users"},
			expected: true,
		},
		{
			name:     "method matches but URL does not",
			request:  NetworkRequest{URL: "https://example.com/api/posts", Method: "POST"},
			matcher:  RequestMatcher{Method: "POST", URLContains: "/api/users"},
			expected: false,
		},
		{
			name:     "URL matches but method does not",
			request:  NetworkRequest{URL: "https://example.com/api/users", Method: "GET"},
			matcher:  RequestMatcher{Method: "POST", URLContains: "/api/users"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matchesRequest(tc.request, tc.matcher)
			if result != tc.expected {
				t.Errorf("matchesRequest(%+v, %+v) = %v, expected %v", tc.request, tc.matcher, result, tc.expected)
			}
		})
	}
}

func TestNewNetworkTracker(t *testing.T) {
	tracker := NewNetworkTracker()
	if tracker == nil {
		t.Fatal("NewNetworkTracker() returned nil")
	}
	if tracker.requests == nil {
		t.Error("requests slice should be initialised")
	}
	if len(tracker.requests) != 0 {
		t.Errorf("requests should be empty, got %d", len(tracker.requests))
	}
}

func TestNewRequestInterceptor(t *testing.T) {
	interceptor := NewRequestInterceptor()
	if interceptor == nil {
		t.Fatal("NewRequestInterceptor() returned nil")
	}
	if interceptor.mocks == nil {
		t.Error("mocks map should be initialised")
	}
	if interceptor.patterns == nil {
		t.Error("patterns slice should be initialised")
	}
	if interceptor.enabled {
		t.Error("interceptor should be disabled by default")
	}
}

func TestRequestInterceptor_AddMock(t *testing.T) {
	ri := NewRequestInterceptor()
	mock := MockResponse{
		StatusCode: 200,
		Body:       `{"ok": true}`,
		Headers:    map[string]string{"Content-Type": "application/json"},
	}

	ri.AddMock("/api/test", mock)

	if len(ri.mocks) != 1 {
		t.Fatalf("expected 1 mock, got %d", len(ri.mocks))
	}
	if len(ri.patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(ri.patterns))
	}
	if ri.patterns[0] != "/api/test" {
		t.Errorf("pattern = %q, expected %q", ri.patterns[0], "/api/test")
	}

	stored := ri.mocks["/api/test"]
	if stored.StatusCode != 200 {
		t.Errorf("StatusCode = %d, expected 200", stored.StatusCode)
	}
	if stored.Body != `{"ok": true}` {
		t.Errorf("Body = %q, expected %q", stored.Body, `{"ok": true}`)
	}
}

func TestRequestInterceptor_RemoveMock(t *testing.T) {
	ri := NewRequestInterceptor()
	ri.AddMock("/api/a", MockResponse{StatusCode: 200})
	ri.AddMock("/api/b", MockResponse{StatusCode: 201})

	ri.RemoveMock("/api/a")

	if len(ri.mocks) != 1 {
		t.Fatalf("expected 1 mock after removal, got %d", len(ri.mocks))
	}
	if _, ok := ri.mocks["/api/a"]; ok {
		t.Error("mock /api/a should have been removed")
	}
	if len(ri.patterns) != 1 {
		t.Fatalf("expected 1 pattern after removal, got %d", len(ri.patterns))
	}
	if ri.patterns[0] != "/api/b" {
		t.Errorf("remaining pattern = %q, expected %q", ri.patterns[0], "/api/b")
	}
}

func TestRequestInterceptor_RemoveMock_NonExistent(t *testing.T) {
	ri := NewRequestInterceptor()
	ri.AddMock("/api/a", MockResponse{StatusCode: 200})

	ri.RemoveMock("/api/nonexistent")

	if len(ri.mocks) != 1 {
		t.Fatalf("expected 1 mock, got %d", len(ri.mocks))
	}
}

func TestParseTrackedRequestEntry(t *testing.T) {
	testCases := []struct {
		name       string
		input      map[string]any
		wantURL    string
		wantMethod string
		wantStatus int
	}{
		{
			name: "complete entry",
			input: map[string]any{
				"url":    "https://example.com/api",
				"method": "POST",
				"status": float64(201),
			},
			wantURL:    "https://example.com/api",
			wantMethod: "POST",
			wantStatus: 201,
		},
		{
			name:       "empty map",
			input:      map[string]any{},
			wantURL:    "",
			wantMethod: "",
			wantStatus: 0,
		},
		{
			name: "missing status",
			input: map[string]any{
				"url":    "https://example.com",
				"method": "GET",
			},
			wantURL:    "https://example.com",
			wantMethod: "GET",
			wantStatus: 0,
		},
		{
			name: "non-float64 status",
			input: map[string]any{
				"url":    "https://example.com",
				"method": "GET",
				"status": "200",
			},
			wantURL:    "https://example.com",
			wantMethod: "GET",
			wantStatus: 0,
		},
		{
			name: "non-string url",
			input: map[string]any{
				"url":    42,
				"method": "GET",
				"status": float64(200),
			},
			wantURL:    "",
			wantMethod: "GET",
			wantStatus: 200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := parseTrackedRequestEntry(tc.input)
			if request.URL != tc.wantURL {
				t.Errorf("URL = %q, expected %q", request.URL, tc.wantURL)
			}
			if request.Method != tc.wantMethod {
				t.Errorf("Method = %q, expected %q", request.Method, tc.wantMethod)
			}
			if request.StatusCode != tc.wantStatus {
				t.Errorf("StatusCode = %d, expected %d", request.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestRequestInterceptor_ClearMocks(t *testing.T) {
	ri := NewRequestInterceptor()
	ri.AddMock("/api/a", MockResponse{StatusCode: 200})
	ri.AddMock("/api/b", MockResponse{StatusCode: 201})
	ri.AddMock("/api/c", MockResponse{StatusCode: 202})

	ri.ClearMocks()

	if len(ri.mocks) != 0 {
		t.Errorf("expected 0 mocks after clear, got %d", len(ri.mocks))
	}
	if len(ri.patterns) != 0 {
		t.Errorf("expected 0 patterns after clear, got %d", len(ri.patterns))
	}
}
