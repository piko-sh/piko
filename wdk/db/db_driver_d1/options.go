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
	"time"
)

const (
	// defaultRequestTimeout bounds a single D1 API call, including the wait for the shared
	// rate limiter.
	defaultRequestTimeout = 2 * time.Minute

	// defaultMaxResponseBytes caps the size of a single D1 API response body at 64 MiB.
	defaultMaxResponseBytes int64 = 64 << 20

	// defaultRequestsPerSecond matches the Cloudflare API's documented account limit of 1200
	// requests per five minutes.
	defaultRequestsPerSecond = 4.0
)

// Option configures a D1 database handle opened through Open.
type Option func(*clientOptions)

// clientOptions holds the tunable limits applied to the HTTP client shared by every
// connection of one database handle.
type clientOptions struct {
	// baseURL overrides the Cloudflare API base URL; empty selects the library default.
	baseURL string

	// requestTimeout bounds each D1 API call, including the wait for the shared rate
	// limiter.
	requestTimeout time.Duration

	// maxResponseBytes is the largest response body accepted before the request fails.
	maxResponseBytes int64

	// requestsPerSecond is the rate shared by every connection of the database handle.
	requestsPerSecond float64
}

// WithRequestTimeout sets how long a single D1 API call may take.
//
// The timeout covers the wait for the shared rate limiter and the HTTP request, and
// applies alongside any deadline on the caller's context. The default is generous so slow
// queries are unaffected; values of zero or below are ignored.
//
// Takes timeout (time.Duration) which is the maximum duration of one call.
//
// Returns Option which applies the timeout.
func WithRequestTimeout(timeout time.Duration) Option {
	return func(options *clientOptions) {
		if timeout > 0 {
			options.requestTimeout = timeout
		}
	}
}

// WithMaxResponseBytes sets the largest D1 API response body the driver accepts.
//
// A larger response fails the call with ErrResponseTooLarge rather than being read into
// memory. Values of zero or below are ignored.
//
// Takes limit (int64) which is the maximum response size in bytes.
//
// Returns Option which applies the limit.
func WithMaxResponseBytes(limit int64) Option {
	return func(options *clientOptions) {
		if limit > 0 {
			options.maxResponseBytes = limit
		}
	}
}

// WithRequestsPerSecond sets the request rate shared by every connection of the database
// handle.
//
// Values of zero or below are ignored, leaving the Cloudflare API's documented default in
// force.
//
// Takes rate (float64) which is the permitted number of requests per second.
//
// Returns Option which applies the rate.
func WithRequestsPerSecond(rate float64) Option {
	return func(options *clientOptions) {
		if rate > 0 {
			options.requestsPerSecond = rate
		}
	}
}

// withBaseURL points the client at a different API base URL.
//
// Takes baseURL (string) which is the base URL to send requests to.
//
// Returns Option which applies the base URL.
func withBaseURL(baseURL string) Option {
	return func(options *clientOptions) {
		options.baseURL = baseURL
	}
}

// newClientOptions returns the defaults with every supplied option applied in order.
//
// Takes options ([]Option) which override the defaults.
//
// Returns clientOptions which is the resolved configuration.
func newClientOptions(options []Option) clientOptions {
	resolved := clientOptions{
		baseURL:           "",
		requestTimeout:    defaultRequestTimeout,
		maxResponseBytes:  defaultMaxResponseBytes,
		requestsPerSecond: defaultRequestsPerSecond,
	}
	for _, option := range options {
		option(&resolved)
	}
	return resolved
}
