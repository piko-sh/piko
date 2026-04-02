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

package storage_domain

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"
)

const (
	// DefaultPresignExpiry is the default length of time a token stays valid.
	DefaultPresignExpiry = 15 * time.Minute

	// MaxPresignExpiry is the maximum allowed validity duration for a
	// presigned token.
	MaxPresignExpiry = 1 * time.Hour

	// DefaultPresignMaxSize is the default largest file size allowed for uploads
	// (100 MB).
	DefaultPresignMaxSize = 100 * 1024 * 1024

	// MaxPresignMaxSize is the highest allowed upload size, set to 1 GB.
	MaxPresignMaxSize = 1024 * 1024 * 1024

	// DefaultPresignRateLimit is the default rate limit for uploads per IP address
	// per minute.
	DefaultPresignRateLimit = 50

	// presignSecretKeyLength is the length of auto-generated secrets.
	presignSecretKeyLength = 32
)

// PresignConfig holds configuration for service-level presigned URLs.
// This is used when a storage provider does not support native presigned URLs.
type PresignConfig struct {
	// RIDCache holds the cache for replay protection using random identifiers.
	// Initialised automatically during service construction via EnsureRIDCache.
	RIDCache *PresignRIDCache

	// BaseURL is the base URL for creating presigned upload URLs.
	// If empty, the URL is created relative to the request origin.
	BaseURL string

	// Secret is the HMAC secret for signing tokens (minimum 32 bytes).
	// If empty, a random secret is generated on first use.
	Secret []byte

	// DefaultExpiry is the default duration for token validity.
	// Must not exceed MaxExpiry.
	DefaultExpiry time.Duration

	// MaxExpiry is the maximum allowed token validity duration.
	// Requests for longer expiry are capped to this value.
	MaxExpiry time.Duration

	// DefaultMaxSize is the default maximum upload size in bytes.
	// Individual presign requests can specify smaller limits.
	DefaultMaxSize int64

	// MaxMaxSize is the absolute maximum upload size in bytes.
	// Requests for larger limits are capped to this value.
	MaxMaxSize int64

	// RateLimitPerMinute is the per-IP rate limit for upload requests.
	// Set to 0 to disable rate limiting.
	RateLimitPerMinute int
}

// Validate checks the configuration and applies defaults where needed.
//
// Returns error when the configuration contains invalid values that cannot
// be corrected automatically.
func (c *PresignConfig) Validate() error {
	if len(c.Secret) > 0 && len(c.Secret) < presignSecretMinLength {
		return fmt.Errorf("presign: secret must be at least %d bytes, got %d", presignSecretMinLength, len(c.Secret))
	}

	if c.DefaultExpiry <= 0 {
		c.DefaultExpiry = DefaultPresignExpiry
	}

	if c.MaxExpiry <= 0 {
		c.MaxExpiry = MaxPresignExpiry
	}

	if c.DefaultExpiry > c.MaxExpiry {
		c.DefaultExpiry = c.MaxExpiry
	}

	if c.DefaultMaxSize <= 0 {
		c.DefaultMaxSize = DefaultPresignMaxSize
	}

	if c.MaxMaxSize <= 0 {
		c.MaxMaxSize = MaxPresignMaxSize
	}

	if c.DefaultMaxSize > c.MaxMaxSize {
		c.DefaultMaxSize = c.MaxMaxSize
	}

	if c.RateLimitPerMinute < 0 {
		c.RateLimitPerMinute = 0
	}

	return nil
}

// EnsureSecret generates a random secret if one is not already set.
//
// Returns error when random generation fails.
func (c *PresignConfig) EnsureSecret() error {
	if len(c.Secret) >= presignSecretMinLength {
		return nil
	}

	secret := make([]byte, presignSecretKeyLength)
	if _, err := rand.Read(secret); err != nil {
		return fmt.Errorf("presign: failed to generate secret: %w", err)
	}
	c.Secret = secret
	return nil
}

// ClampExpiry constrains the expiry duration to allowed bounds.
//
// Takes expiry (time.Duration) which is the requested expiry duration.
//
// Returns time.Duration which is the clamped expiry value.
func (c *PresignConfig) ClampExpiry(expiry time.Duration) time.Duration {
	if expiry <= 0 {
		return c.DefaultExpiry
	}
	if expiry > c.MaxExpiry {
		return c.MaxExpiry
	}
	return expiry
}

// ClampMaxSize constrains the max size to allowed bounds.
//
// Takes maxSize (int64) which is the requested maximum size.
//
// Returns int64 which is the clamped max size value.
func (c *PresignConfig) ClampMaxSize(maxSize int64) int64 {
	if maxSize <= 0 {
		return c.DefaultMaxSize
	}
	if maxSize > c.MaxMaxSize {
		return c.MaxMaxSize
	}
	return maxSize
}

// EnsureRIDCache initialises the random identifier cache if not already set.
//
// Takes ctx (context.Context) which is the parent context for background
// goroutines.
// Takes cleanupInterval (time.Duration) which specifies how often to purge
// expired identifiers.
func (c *PresignConfig) EnsureRIDCache(ctx context.Context, cleanupInterval time.Duration) {
	if c.RIDCache == nil {
		c.RIDCache = NewPresignRIDCache(ctx, cleanupInterval)
	}
}

// DefaultPresignConfig returns a PresignConfig with sensible default values.
//
// Returns PresignConfig which is ready for use with default settings.
func DefaultPresignConfig() PresignConfig {
	return PresignConfig{
		Secret:             nil,
		DefaultExpiry:      DefaultPresignExpiry,
		MaxExpiry:          MaxPresignExpiry,
		DefaultMaxSize:     DefaultPresignMaxSize,
		MaxMaxSize:         MaxPresignMaxSize,
		RateLimitPerMinute: DefaultPresignRateLimit,
		BaseURL:            "",
		RIDCache:           nil,
	}
}
