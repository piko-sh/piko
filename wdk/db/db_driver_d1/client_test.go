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

package db_driver_d1

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRequestsPerSecond = 1000
)

type recordingServer struct {
	server   *httptest.Server
	requests []recordedRequest
	mu       sync.Mutex
	count    atomic.Int64
}

type recordedRequest struct {
	path string
	body rawQueryRequest
}

func newRecordingServer(t *testing.T, status int, responseBody string) *recordingServer {
	t.Helper()

	recorder := &recordingServer{}
	recorder.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		recorder.count.Add(1)
		payload, err := io.ReadAll(request.Body)
		if !assert.NoError(t, err) {
			return
		}
		var body rawQueryRequest
		assert.NoError(t, json.Unmarshal(payload, &body))
		recorder.mu.Lock()
		recorder.requests = append(recorder.requests, recordedRequest{path: request.URL.Path, body: body})
		recorder.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(recorder.server.Close)
	return recorder
}

func (r *recordingServer) recorded() []recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]recordedRequest(nil), r.requests...)
}

func newTestClient(t *testing.T, serverURL string, options ...Option) *d1Client {
	t.Helper()

	allOptions := append([]Option{withBaseURL(serverURL), WithRequestsPerSecond(testRequestsPerSecond)}, options...)
	client, err := newD1Client(Config{
		APIToken:   "test-token",
		AccountID:  "test-account",
		DatabaseID: "test-database",
	}, newClientOptions(allOptions))
	require.NoError(t, err)
	t.Cleanup(client.close)
	return client
}

func newTestConn(t *testing.T, responseBody string) *d1Conn {
	t.Helper()

	recorder := newRecordingServer(t, http.StatusOK, responseBody)
	return newConn(newTestClient(t, recorder.server.URL), nil)
}

func TestClientQuerySendsRawRequest(t *testing.T) {
	recorder := newRecordingServer(t, http.StatusOK, singleSuccessResult)
	client := newTestClient(t, recorder.server.URL)

	results, err := client.query(context.Background(), "SELECT ?", nil)
	require.NoError(t, err)
	require.Len(t, results, 1)

	requests := recorder.recorded()
	require.Len(t, requests, 1)
	assert.Equal(t, "/accounts/test-account/d1/database/test-database/raw", requests[0].path)
	assert.Equal(t, "SELECT ?", requests[0].body.SQL)
	assert.Equal(t, []string{}, requests[0].body.Parameters)
}

func TestClientQueryFailures(t *testing.T) {
	testCases := []struct {
		name          string
		responseBody  string
		expectedError string
		status        int
	}{
		{
			name:          "api reports failure with messages",
			status:        http.StatusOK,
			responseBody:  `{"result": null, "success": false, "errors": [{"code": 7500, "message": "syntax error"}], "messages": []}`,
			expectedError: "7500: syntax error",
		},
		{
			name:          "api reports failure without messages",
			status:        http.StatusOK,
			responseBody:  `{"result": null, "success": false, "errors": [], "messages": []}`,
			expectedError: "without an error message",
		},
		{
			name:          "result is not an array",
			status:        http.StatusOK,
			responseBody:  `{"result": {"unexpected": true}, "success": true, "errors": [], "messages": []}`,
			expectedError: "decoding D1 results",
		},
		{
			name:          "statement reports failure",
			status:        http.StatusOK,
			responseBody:  firstSuccessSecondFailure,
			expectedError: "statement 1: D1 query returned failure",
		},
		{
			name:          "request rejected with client error",
			status:        http.StatusBadRequest,
			responseBody:  `{"result": null, "success": false, "errors": [{"code": 7500, "message": "no such table: t"}], "messages": []}`,
			expectedError: "no such table: t",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := newRecordingServer(t, testCase.status, testCase.responseBody)
			client := newTestClient(t, recorder.server.URL)

			results, err := client.query(context.Background(), "SELECT 1", nil)
			require.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), testCase.expectedError)
		})
	}
}

func TestClientQueryIsNeverResent(t *testing.T) {
	testCases := []struct {
		name   string
		status int
	}{
		{name: "bad gateway", status: http.StatusBadGateway},
		{name: "service unavailable", status: http.StatusServiceUnavailable},
		{name: "too many requests", status: http.StatusTooManyRequests},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := newRecordingServer(t, testCase.status, `{"success": false}`)
			client := newTestClient(t, recorder.server.URL)

			_, err := client.query(context.Background(), "INSERT INTO t (v) VALUES (?)", []string{"x"})
			require.Error(t, err)
			assert.Equal(t, int64(1), recorder.count.Load())
		})
	}
}

func TestClientQueryAppliesRequestTimeout(t *testing.T) {
	recorder := newRecordingServer(t, http.StatusOK, singleSuccessResult)
	client := newTestClient(t, recorder.server.URL, WithRequestTimeout(time.Nanosecond))

	_, err := client.query(context.Background(), "SELECT 1", nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestNewD1ClientRejectsEmptyToken(t *testing.T) {
	client, err := newD1Client(Config{APIToken: "", AccountID: "a", DatabaseID: "d"}, newClientOptions(nil))
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "creating API client")
}

func TestNewD1ClientEscapesIdentifiersInEndpoint(t *testing.T) {
	client, err := newD1Client(Config{APIToken: "token", AccountID: "a/b", DatabaseID: "d?x"}, newClientOptions(nil))
	require.NoError(t, err)
	t.Cleanup(client.close)

	assert.Equal(t, "/accounts/a%2Fb/d1/database/d%3Fx/raw", client.endpoint)
}

func TestDescribeAPIFailure(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		failures []cloudflare.ResponseInfo
	}{
		{
			name:     "no failures",
			failures: nil,
			expected: "D1 API reported failure without an error message",
		},
		{
			name: "several failures",
			failures: []cloudflare.ResponseInfo{
				{Code: 1, Message: "first"},
				{Code: 2, Message: "second"},
			},
			expected: "D1 API reported failure: 1: first; 2: second",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.EqualError(t, describeAPIFailure(testCase.failures), testCase.expected)
		})
	}
}

func TestDecodeStatementResultsKeepsLargeIntegers(t *testing.T) {
	results, err := decodeStatementResults([]byte(`[{"success": true, "results": {"columns": ["big"], "rows": [[9007199254740993]]}, "meta": {"last_row_id": 9007199254740993, "changes": 1}}]`))
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, json.Number("9007199254740993"), results[0].Results.Rows[0][0])
	assert.Equal(t, int64(9007199254740993), results[0].Meta.LastRowID)
}
