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
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var (
	// ErrResponseTooLarge reports that a D1 API response body exceeded the configured
	// maximum size. Callers can match it with errors.Is.
	ErrResponseTooLarge = errors.New("db_driver_d1: D1 API response exceeds the maximum size")
)

// idleConnectionCloser is implemented by transports that can release their idle
// connections.
type idleConnectionCloser interface {
	// CloseIdleConnections closes every idle connection the transport holds.
	CloseIdleConnections()
}

// boundedBodyTransport reads every response body into memory up to a fixed limit before
// handing the response on, so the Cloudflare client never reads an unbounded body and the
// underlying connection is always drained and closed, even for error statuses the client
// abandons without closing.
type boundedBodyTransport struct {
	// base performs the network round trip.
	base http.RoundTripper

	// maxResponseBytes is the largest body accepted; a larger one fails the round trip.
	maxResponseBytes int64
}

// RoundTrip performs the request and replaces the response body with a bounded in-memory
// copy.
//
// Takes request (*http.Request) which is the outgoing API request.
//
// Returns *http.Response which carries the buffered body.
// Returns error when the round trip fails, the body cannot be read, or the body exceeds
// the configured limit (wrapping ErrResponseTooLarge).
func (t *boundedBodyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(request)
	if err != nil {
		return nil, err
	}

	body, err := readBoundedBody(response.Body, t.maxResponseBytes)
	if err != nil {
		return nil, fmt.Errorf("reading D1 API response (HTTP %d): %w", response.StatusCode, err)
	}

	response.Body = io.NopCloser(bytes.NewReader(body))
	response.ContentLength = int64(len(body))
	return response, nil
}

// CloseIdleConnections closes the idle connections held by the base transport.
func (t *boundedBodyTransport) CloseIdleConnections() {
	if closer, ok := t.base.(idleConnectionCloser); ok {
		closer.CloseIdleConnections()
	}
}

// newBaseTransport returns a private copy of the default HTTP transport so the idle
// connections of one database handle can be closed without affecting other clients.
//
// Returns http.RoundTripper which is the transport used for API requests.
func newBaseTransport() http.RoundTripper {
	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		return transport.Clone()
	}
	return &http.Transport{Proxy: http.ProxyFromEnvironment}
}

// readBoundedBody reads body to completion, failing when it holds more than limit bytes,
// and always closes it.
//
// Reading limit+1 bytes detects an oversized body without reading the rest of it; a body
// within the limit is read to EOF so the connection can be reused.
//
// Takes body (io.ReadCloser) which is the response body to consume.
// Takes limit (int64) which is the largest acceptable size in bytes.
//
// Returns []byte which is the complete body.
// Returns error when reading fails or the body exceeds limit (wrapping
// ErrResponseTooLarge).
func readBoundedBody(body io.ReadCloser, limit int64) ([]byte, error) {
	data, readErr := io.ReadAll(io.LimitReader(body, limit+1))
	closeErr := body.Close()
	if readErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w (limit %d bytes)", ErrResponseTooLarge, limit)
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return data, nil
}
