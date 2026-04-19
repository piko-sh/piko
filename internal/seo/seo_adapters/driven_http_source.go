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
	"io"
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

	// defaultMaxSitemapResponseBytes caps the raw response body size accepted
	// from a dynamic sitemap source.
	//
	// Without a cap, a hostile or misconfigured endpoint could exhaust memory by
	// streaming an unbounded JSON array. Sixteen mebibytes comfortably accommodates
	// legitimate sitemap responses while preventing denial-of-service.
	defaultMaxSitemapResponseBytes int64 = 16 * 1024 * 1024
)

// ErrSitemapResponseTooLarge signals that the dynamic sitemap source returned
// a response whose size exceeded the configured maximum. Callers can use
// errors.Is to distinguish this from network or decoding failures.
var ErrSitemapResponseTooLarge = errors.New("sitemap source response exceeded maximum allowed size")

// HTTPSourceAdapter implements the DynamicURLSourcePort interface by fetching
// URLs from HTTP endpoints. It expects endpoints to return JSON arrays of
// SitemapURLInput objects.
type HTTPSourceAdapter struct {
	// httpClient sends HTTP requests to fetch sitemaps and other resources.
	httpClient *http.Client

	// breaker guards against failures when fetching from the HTTP source.
	breaker *gobreaker.CircuitBreaker[[]seo_dto.SitemapURLInput]

	// maxResponseBytes caps the response body size accepted from a dynamic sitemap source.
	//
	// Bodies larger than this limit cause FetchURLs to return
	// ErrSitemapResponseTooLarge instead of attempting to decode them. A
	// non-positive value disables the cap.
	maxResponseBytes int64
}

// HTTPSourceOption configures an HTTPSourceAdapter at construction time.
type HTTPSourceOption func(*HTTPSourceAdapter)

// WithHTTPSourceMaxResponseBytes overrides the maximum response body size
// accepted from the dynamic sitemap source. Pass zero or a negative value to
// disable the cap entirely.
//
// Takes maxBytes (int64) which is the new ceiling, in bytes, for responses
// from sitemap source endpoints.
//
// Returns HTTPSourceOption which applies the override during construction.
func WithHTTPSourceMaxResponseBytes(maxBytes int64) HTTPSourceOption {
	return func(a *HTTPSourceAdapter) {
		a.maxResponseBytes = maxBytes
	}
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
		defer func() {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}()

		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected HTTP status code: %d %s", response.StatusCode, response.Status)
		}

		body, err := a.readBoundedBody(response.Body)
		if err != nil {
			return nil, err
		}

		var urls []seo_dto.SitemapURLInput
		if err := json.ConfigStd.Unmarshal(body, &urls); err != nil {
			return nil, fmt.Errorf("decoding JSON response: %w", err)
		}

		return urls, nil
	})
}

// readBoundedBody reads the response body up to the configured maximum size.
// Bodies that exceed the limit return a wrapped ErrSitemapResponseTooLarge so
// callers can distinguish over-large responses from transport failures.
//
// Takes body (io.Reader) which yields the raw HTTP response body bytes.
//
// Returns []byte which contains the raw body bytes when the cap was not
// exceeded.
// Returns error when reading fails or the body exceeded the configured cap.
func (a *HTTPSourceAdapter) readBoundedBody(body io.Reader) ([]byte, error) {
	maxBytes := a.maxResponseBytes
	if maxBytes <= 0 {
		raw, err := io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("reading sitemap source response body: %w", err)
		}
		return raw, nil
	}

	raw, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading sitemap source response body: %w", err)
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("%w: limit=%d bytes", ErrSitemapResponseTooLarge, maxBytes)
	}
	return raw, nil
}

// NewHTTPSourceAdapter creates a new HTTP source adapter with sensible
// defaults.
//
// Takes opts (...HTTPSourceOption) which override defaults such as the
// maximum response body size accepted from the sitemap endpoint.
//
// Returns seo_domain.DynamicURLSourcePort which is the configured adapter ready
// for use.
func NewHTTPSourceAdapter(opts ...HTTPSourceOption) seo_domain.DynamicURLSourcePort {
	adapter := &HTTPSourceAdapter{
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
		breaker:          newHTTPSourceCircuitBreaker(),
		maxResponseBytes: defaultMaxSitemapResponseBytes,
	}

	for _, opt := range opts {
		opt(adapter)
	}

	return adapter
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
