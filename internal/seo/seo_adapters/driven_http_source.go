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

package seo_adapters

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"piko.sh/piko/internal/json"
	"github.com/sony/gobreaker/v2"
	"piko.sh/piko/internal/seo/seo_domain"
	"piko.sh/piko/internal/seo/seo_dto"
)

const (
	// defaultHTTPTimeout is the default timeout for HTTP requests to dynamic URL
	// sources.
	defaultHTTPTimeout = 30 * time.Second

	// defaultMaxIdleConns is the default limit for idle HTTP connections.
	defaultMaxIdleConns = 10

	// defaultIdleConnTimeout is how long an idle HTTP connection may stay open.
	defaultIdleConnTimeout = 90 * time.Second

	// defaultMaxIdleConnsPerHost is the default maximum number of idle connections
	// per host.
	defaultMaxIdleConnsPerHost = 10

	// circuitBreakerTimeout is the duration the circuit stays open before
	// allowing a test request through.
	circuitBreakerTimeout = 30 * time.Second

	// circuitBreakerBucketPeriod is the duration of each measurement bucket
	// for tracking failure counts.
	circuitBreakerBucketPeriod = 10 * time.Second

	// circuitBreakerConsecutiveFailures is the number of consecutive failures
	// required to trip the circuit breaker.
	circuitBreakerConsecutiveFailures = 5
)

// HTTPSourceAdapter implements the DynamicURLSourcePort interface by fetching
// URLs from HTTP endpoints. It expects endpoints to return JSON arrays of
// SitemapURLInput objects.
type HTTPSourceAdapter struct {
	// httpClient sends HTTP requests to fetch sitemaps and other resources.
	httpClient *http.Client

	// breaker guards against failures when fetching from the HTTP source.
	breaker *gobreaker.CircuitBreaker[[]seo_dto.SitemapURLInput]
}

// FetchURLs implements DynamicURLSourcePort.FetchURLs.
// It makes an HTTP GET request to the source URL and decodes the JSON response.
//
// Takes sourceURL (string) which specifies the endpoint to fetch URLs from.
//
// Returns []seo_dto.SitemapURLInput which contains the decoded URL entries.
// Returns error when the request fails, the response cannot be decoded, or the
// circuit breaker is open.
func (a *HTTPSourceAdapter) FetchURLs(ctx context.Context, sourceURL string) ([]seo_dto.SitemapURLInput, error) {
	return a.breaker.Execute(func() ([]seo_dto.SitemapURLInput, error) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating HTTP request: %w", err)
		}

		request.Header.Set("Accept", "application/json")
		request.Header.Set("User-Agent", "Piko-SEO-Service/1.0")

		response, err := a.httpClient.Do(request)
		if err != nil {
			return nil, fmt.Errorf("executing HTTP request: %w", err)
		}
		defer func() { _ = response.Body.Close() }()

		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected HTTP status code: %d %s", response.StatusCode, response.Status)
		}

		var urls []seo_dto.SitemapURLInput
		decoder := json.ConfigStd.NewDecoder(response.Body)
		if err := decoder.Decode(&urls); err != nil {
			return nil, fmt.Errorf("decoding JSON response: %w", err)
		}

		return urls, nil
	})
}

// NewHTTPSourceAdapter creates a new HTTP source adapter with sensible
// defaults.
//
// Returns seo_domain.DynamicURLSourcePort which is the configured adapter ready
// for use.
func NewHTTPSourceAdapter() seo_domain.DynamicURLSourcePort {
	return &HTTPSourceAdapter{
		httpClient: &http.Client{
			Timeout:       defaultHTTPTimeout,
			CheckRedirect: nil,
			Jar:           nil,
			Transport: &http.Transport{
				Proxy:                  nil,
				OnProxyConnectResponse: nil,
				DialContext:            nil,
				DialTLSContext:         nil,
				TLSClientConfig:        nil,
				TLSHandshakeTimeout:    0,
				DisableKeepAlives:      false,
				DisableCompression:     false,
				MaxIdleConns:           defaultMaxIdleConns,
				MaxIdleConnsPerHost:    defaultMaxIdleConnsPerHost,
				MaxConnsPerHost:        0,
				IdleConnTimeout:        defaultIdleConnTimeout,
				ResponseHeaderTimeout:  0,
				ExpectContinueTimeout:  0,
				TLSNextProto:           nil,
				ProxyConnectHeader:     nil,
				GetProxyConnectHeader:  nil,
				MaxResponseHeaderBytes: 0,
				WriteBufferSize:        0,
				ReadBufferSize:         0,
				ForceAttemptHTTP2:      false,
				HTTP2:                  nil,
				Protocols:              nil,
			},
		},
		breaker: newHTTPSourceCircuitBreaker(),
	}
}

// newHTTPSourceCircuitBreaker creates a circuit breaker for the HTTP source
// adapter.
//
// Returns *gobreaker.CircuitBreaker[[]seo_dto.SitemapURLInput] configured with
// standard settings for HTTP source operations.
func newHTTPSourceCircuitBreaker() *gobreaker.CircuitBreaker[[]seo_dto.SitemapURLInput] {
	settings := gobreaker.Settings{
		Name:         "seo-http-source",
		MaxRequests:  1,
		Interval:     0,
		Timeout:      circuitBreakerTimeout,
		BucketPeriod: circuitBreakerBucketPeriod,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= circuitBreakerConsecutiveFailures
		},
		IsExcluded: func(err error) bool {
			return errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded)
		},
	}
	return gobreaker.NewCircuitBreaker[[]seo_dto.SitemapURLInput](settings)
}
