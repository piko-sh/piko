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

package templater_dto

import (
	"context"
	"net/http"
	"sync"
)

// cookieAccumulatorKey is the context key for storing the CookieAccumulator.
type cookieAccumulatorKey struct{}

// CookieAccumulator collects response cookies during rendering. Both pages
// and nested partials can add cookies through the shared accumulator in the
// request context.
//
// Safe for concurrent use by multiple goroutines.
type CookieAccumulator struct {
	// cookies holds the accumulated response cookies.
	cookies []*http.Cookie

	// mu guards concurrent access to cookies.
	mu sync.Mutex
}

// NewCookieAccumulator creates a new CookieAccumulator ready for use.
//
// Returns *CookieAccumulator which is ready to collect cookies.
func NewCookieAccumulator() *CookieAccumulator {
	return &CookieAccumulator{}
}

// Add appends a cookie to the accumulator.
//
// Takes cookie (*http.Cookie) which specifies the cookie to add to the
// response.
//
// Safe for concurrent use.
func (a *CookieAccumulator) Add(cookie *http.Cookie) {
	if cookie == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cookies = append(a.cookies, cookie)
}

// GetCookies returns all accumulated cookies as a defensive copy.
//
// Returns []*http.Cookie which contains all cookies added via Add.
//
// Safe for concurrent use.
func (a *CookieAccumulator) GetCookies() []*http.Cookie {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]*http.Cookie, len(a.cookies))
	copy(result, a.cookies)
	return result
}

// WithCookieAccumulator returns a new context with the given CookieAccumulator
// attached.
//
// Takes ctx (context.Context) which is the parent context.
// Takes acc (*CookieAccumulator) which is the accumulator to attach.
//
// Returns context.Context which contains the accumulator.
func WithCookieAccumulator(ctx context.Context, acc *CookieAccumulator) context.Context {
	return context.WithValue(ctx, cookieAccumulatorKey{}, acc)
}

// CookieAccumulatorFromContext retrieves the CookieAccumulator from the
// context, or nil if none is present.
//
// Takes ctx (context.Context) which may contain a CookieAccumulator.
//
// Returns *CookieAccumulator which is the accumulator, or nil if not found.
func CookieAccumulatorFromContext(ctx context.Context) *CookieAccumulator {
	if ctx == nil {
		return nil
	}
	acc, ok := ctx.Value(cookieAccumulatorKey{}).(*CookieAccumulator)
	if !ok {
		return nil
	}
	return acc
}
