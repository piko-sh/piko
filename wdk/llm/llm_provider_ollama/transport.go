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

package llm_provider_ollama

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	// responseDrainLimit caps how much of an unread response body is discarded before the
	// body is closed, so the connection can be reused without reading unbounded data.
	responseDrainLimit = 64 << 10

	// fallbackDialTimeout bounds establishing a connection with the fallback transport.
	fallbackDialTimeout = 30 * time.Second

	// fallbackKeepAlive is the TCP keep-alive period of the fallback transport.
	fallbackKeepAlive = 30 * time.Second

	// fallbackMaxIdleConnections caps the idle connections pooled by the fallback transport.
	fallbackMaxIdleConnections = 100

	// fallbackIdleConnTimeout is how long the fallback transport keeps idle connections.
	fallbackIdleConnTimeout = 90 * time.Second

	// fallbackTLSHandshakeTimeout bounds TLS handshakes made by the fallback transport.
	fallbackTLSHandshakeTimeout = 10 * time.Second

	// fallbackExpectContinueTimeout bounds the wait for a 100-continue response.
	fallbackExpectContinueTimeout = time.Second
)

// cloneTransport returns a private copy of base when it is an *http.Transport.
//
// Applications may replace http.DefaultTransport with a wrapping round tripper, so the
// fallback mirrors the standard library defaults instead of assuming the concrete type.
//
// Takes base (http.RoundTripper) which is the transport to copy, normally
// http.DefaultTransport.
//
// Returns *http.Transport which is safe to modify without affecting base.
func cloneTransport(base http.RoundTripper) *http.Transport {
	if transport, ok := base.(*http.Transport); ok {
		return transport.Clone()
	}
	return newFallbackTransport()
}

// newFallbackTransport builds a transport with the standard library default settings.
//
// Returns *http.Transport which honours proxy environment variables like the default.
func newFallbackTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   fallbackDialTimeout,
		KeepAlive: fallbackKeepAlive,
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          fallbackMaxIdleConnections,
		IdleConnTimeout:       fallbackIdleConnTimeout,
		TLSHandshakeTimeout:   fallbackTLSHandshakeTimeout,
		ExpectContinueTimeout: fallbackExpectContinueTimeout,
	}
}

// drainAndClose discards a bounded amount of unread body data and closes the body.
//
// Takes body (io.ReadCloser) which is the response body to release.
//
// Returns error which joins any failure to drain or close the body.
func drainAndClose(body io.ReadCloser) error {
	var drainErr error
	if _, err := io.Copy(io.Discard, io.LimitReader(body, responseDrainLimit)); err != nil {
		drainErr = fmt.Errorf("draining response body: %w", err)
	}
	var closeErr error
	if err := body.Close(); err != nil {
		closeErr = fmt.Errorf("closing response body: %w", err)
	}
	return errors.Join(drainErr, closeErr)
}
