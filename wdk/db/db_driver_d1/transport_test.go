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
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type trackingBody struct {
	reader    io.Reader
	readErr   error
	closed    bool
	exhausted bool
}

func (b *trackingBody) Read(buffer []byte) (int, error) {
	if b.readErr != nil {
		return 0, b.readErr
	}
	count, err := b.reader.Read(buffer)
	if errors.Is(err, io.EOF) {
		b.exhausted = true
	}
	return count, err
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}

type fixedRoundTripper struct {
	response   *http.Response
	err        error
	idleClosed bool
}

func (f *fixedRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return f.response, f.err
}

func (f *fixedRoundTripper) CloseIdleConnections() {
	f.idleClosed = true
}

func TestReadBoundedBody(t *testing.T) {
	testCases := []struct {
		readErr       error
		expectedError error
		name          string
		body          string
		limit         int64
		expectDrained bool
	}{
		{name: "below limit", body: "abc", limit: 4, expectDrained: true},
		{name: "exactly at limit", body: "abcd", limit: 4, expectDrained: true},
		{name: "over limit", body: "abcde", limit: 4, expectedError: ErrResponseTooLarge},
		{name: "read failure", readErr: io.ErrUnexpectedEOF, limit: 4, expectedError: io.ErrUnexpectedEOF},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			body := &trackingBody{reader: strings.NewReader(testCase.body), readErr: testCase.readErr}

			data, err := readBoundedBody(body, testCase.limit)
			assert.True(t, body.closed)
			if testCase.expectedError != nil {
				require.ErrorIs(t, err, testCase.expectedError)
				assert.Nil(t, data)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.body, string(data))
			assert.Equal(t, testCase.expectDrained, body.exhausted)
		})
	}
}

func TestBoundedBodyTransportBuffersAndClosesBody(t *testing.T) {
	body := &trackingBody{reader: strings.NewReader("payload")}
	base := &fixedRoundTripper{response: &http.Response{StatusCode: http.StatusBadGateway, Body: body}}
	transport := &boundedBodyTransport{base: base, maxResponseBytes: 16}

	response, err := transport.RoundTrip(&http.Request{})
	require.NoError(t, err)
	assert.True(t, body.closed)
	assert.True(t, body.exhausted)
	assert.Equal(t, int64(len("payload")), response.ContentLength)

	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(data))

	transport.CloseIdleConnections()
	assert.True(t, base.idleClosed)
}

func TestBoundedBodyTransportFailures(t *testing.T) {
	roundTripErr := errors.New("connection refused")
	testCases := []struct {
		expectedError error
		base          *fixedRoundTripper
		name          string
	}{
		{name: "round trip failure", base: &fixedRoundTripper{err: roundTripErr}, expectedError: roundTripErr},
		{
			name: "oversized body",
			base: &fixedRoundTripper{response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       &trackingBody{reader: strings.NewReader(strings.Repeat("x", 32))},
			}},
			expectedError: ErrResponseTooLarge,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			transport := &boundedBodyTransport{base: testCase.base, maxResponseBytes: 16}

			response, err := transport.RoundTrip(&http.Request{})
			require.ErrorIs(t, err, testCase.expectedError)
			assert.Nil(t, response)
		})
	}
}

func TestBoundedBodyTransportIgnoresBaseWithoutIdleClose(t *testing.T) {
	transport := &boundedBodyTransport{base: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("unused")
	}), maxResponseBytes: 1}

	assert.NotPanics(t, transport.CloseIdleConnections)
}

func TestOversizedResponseFailsQuery(t *testing.T) {
	recorder := newRecordingServer(t, http.StatusOK, singleSuccessResult)
	client := newTestClient(t, recorder.server.URL, WithMaxResponseBytes(16))

	_, err := client.query(context.Background(), "SELECT 1", nil)
	require.ErrorIs(t, err, ErrResponseTooLarge)
	assert.Equal(t, int64(1), recorder.count.Load())
}

func TestNewBaseTransportIsPrivate(t *testing.T) {
	first := newBaseTransport()
	second := newBaseTransport()

	assert.NotSame(t, first, second)
	assert.NotSame(t, http.DefaultTransport, first)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
