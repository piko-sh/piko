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

package daemon_domain

import (
	"time"

	"piko.sh/piko/internal/daemon/daemon_dto"
)

const (
	// TransportHTTP is the HTTP transport protocol.
	TransportHTTP Transport = "http"

	// TransportSSE is the Server-Sent Events transport protocol.
	TransportSSE Transport = "sse"
)

// Action is the base interface that all actions must satisfy.
// Actions implement this implicitly by embedding ActionMetadata and
// having a Call method (with any signature).
//
// The actual Call method signature is not part of this interface because
// it varies per action. The parser and code generator handle signature
// validation and wrapper generation.
type Action interface {
	// Ctx returns the request context.
	Ctx() any

	// Request returns the request metadata.
	Request() *daemon_dto.RequestMetadata

	// Response returns the response writer.
	Response() *daemon_dto.ResponseWriter
}

// ResourceLimitable is an interface that actions can implement to configure
// resource limits for protection against denial-of-service and resource
// exhaustion.
type ResourceLimitable interface {
	// ResourceLimits returns the resource limit configuration for this action.
	ResourceLimits() *ResourceLimits
}

// ResourceLimits defines resource constraints for an action.
type ResourceLimits struct {
	// MaxRequestBodySize is the maximum request body size in bytes.
	// A value of 0 uses the default limit.
	MaxRequestBodySize int64

	// MaxResponseSize is the maximum response size in bytes.
	// A value of 0 means no limit.
	MaxResponseSize int64

	// Timeout is the maximum execution time for the action.
	// A value of 0 uses the default timeout.
	Timeout time.Duration

	// SlowThreshold is the duration after which a request is logged as slow.
	// A value of 0 uses the default threshold.
	SlowThreshold time.Duration

	// MaxConcurrent is the maximum concurrent executions of this action.
	// A value of 0 means no limit.
	MaxConcurrent int

	// MaxMemoryUsage is the maximum memory usage in bytes (advisory).
	// A value of 0 means no limit.
	MaxMemoryUsage int64

	// SSE-specific limits
	// MaxSSEDuration is the maximum SSE connection duration.
	MaxSSEDuration time.Duration

	// SSEHeartbeatInterval is the interval between SSE heartbeat messages.
	SSEHeartbeatInterval time.Duration
}

// Cacheable is an interface that actions can implement to configure
// response caching behaviour.
type Cacheable interface {
	// CacheConfig returns the caching configuration for this action.
	CacheConfig() *CacheConfig
}

// CacheConfig defines caching behaviour for an action's responses.
type CacheConfig struct {
	// KeyFunc is an optional function to generate custom cache keys.
	// If nil, the default key is based on action name and arguments.
	KeyFunc func(request *daemon_dto.RequestMetadata, arguments map[string]any) string

	// VaryHeaders lists headers that affect the cache key.
	// Different header values result in different cache entries.
	VaryHeaders []string

	// TTL is the cache time-to-live duration.
	TTL time.Duration
}

// RateLimitable is an interface that actions can implement to configure
// rate limiting for the action.
type RateLimitable interface {
	// RateLimit returns the rate limit configuration for this action.
	RateLimit() *RateLimit
}

// RateLimit defines rate limiting configuration for an action.
type RateLimit struct {
	// KeyFunc determines the rate limit key (e.g., by IP, user, or custom).
	// If nil, defaults to rate limiting by IP address.
	KeyFunc RateLimitKeyFunc

	// RequestsPerMinute is the maximum requests allowed per minute.
	RequestsPerMinute int

	// BurstSize is the maximum burst size for the rate limiter.
	BurstSize int
}

// RateLimitKeyFunc extracts a rate limit key from a request.
type RateLimitKeyFunc func(request *daemon_dto.RequestMetadata) string

var (
	// RateLimitByIP limits requests by client IP address.
	RateLimitByIP RateLimitKeyFunc = func(request *daemon_dto.RequestMetadata) string {
		return request.RemoteAddr
	}

	// RateLimitByUser limits requests by authenticated user ID.
	RateLimitByUser RateLimitKeyFunc = func(request *daemon_dto.RequestMetadata) string {
		if request.Session != nil && request.Session.UserID != "" {
			return request.Session.UserID
		}
		return request.RemoteAddr
	}

	// RateLimitBySession limits requests by session ID.
	RateLimitBySession RateLimitKeyFunc = func(request *daemon_dto.RequestMetadata) string {
		if request.Session != nil && request.Session.ID != "" {
			return request.Session.ID
		}
		return request.RemoteAddr
	}
)

// Transport represents a supported transport mechanism for actions.
type Transport string
