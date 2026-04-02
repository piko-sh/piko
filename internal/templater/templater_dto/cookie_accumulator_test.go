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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCookieAccumulator_Add_GetCookies(t *testing.T) {
	t.Parallel()

	acc := NewCookieAccumulator()

	cookie1 := &http.Cookie{Name: "session", Value: "abc123"}
	cookie2 := &http.Cookie{Name: "pref", Value: "dark"}

	acc.Add(cookie1)
	acc.Add(cookie2)

	cookies := acc.GetCookies()
	require.Len(t, cookies, 2)
	assert.Equal(t, "session", cookies[0].Name)
	assert.Equal(t, "abc123", cookies[0].Value)
	assert.Equal(t, "pref", cookies[1].Name)
	assert.Equal(t, "dark", cookies[1].Value)
}

func TestCookieAccumulator_Add_NilCookie(t *testing.T) {
	t.Parallel()

	acc := NewCookieAccumulator()
	acc.Add(nil)

	cookies := acc.GetCookies()
	assert.Empty(t, cookies)
}

func TestCookieAccumulator_GetCookies_Empty(t *testing.T) {
	t.Parallel()

	acc := NewCookieAccumulator()
	cookies := acc.GetCookies()
	assert.Empty(t, cookies)
	assert.NotNil(t, cookies)
}

func TestCookieAccumulator_GetCookies_DefensiveCopy(t *testing.T) {
	t.Parallel()

	acc := NewCookieAccumulator()
	acc.Add(&http.Cookie{Name: "a", Value: "1"})

	cookies1 := acc.GetCookies()
	cookies2 := acc.GetCookies()

	cookies1[0] = &http.Cookie{Name: "mutated", Value: "x"}
	assert.Equal(t, "a", cookies2[0].Name)
}

func TestCookieAccumulator_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	acc := NewCookieAccumulator()
	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()
			acc.Add(&http.Cookie{
				Name:  "cookie",
				Value: "value",
			})
			_ = idx
		}(i)
	}

	wg.Wait()

	cookies := acc.GetCookies()
	assert.Len(t, cookies, numGoroutines)
}

func TestCookieAccumulator_ContextRoundTrip(t *testing.T) {
	t.Parallel()

	acc := NewCookieAccumulator()
	acc.Add(&http.Cookie{Name: "test", Value: "value"})

	ctx := WithCookieAccumulator(context.Background(), acc)
	retrieved := CookieAccumulatorFromContext(ctx)

	require.NotNil(t, retrieved)
	assert.Same(t, acc, retrieved)

	cookies := retrieved.GetCookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "test", cookies[0].Name)
}

func TestCookieAccumulatorFromContext_NilContext(t *testing.T) {
	t.Parallel()

	acc := CookieAccumulatorFromContext(nil)
	assert.Nil(t, acc)
}

func TestCookieAccumulatorFromContext_NoAccumulator(t *testing.T) {
	t.Parallel()

	acc := CookieAccumulatorFromContext(context.Background())
	assert.Nil(t, acc)
}
