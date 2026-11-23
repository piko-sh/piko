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

package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"piko.sh/json"
)

const (
	// cacheLanguageEnglish is the English language selector.
	cacheLanguageEnglish = "en"

	// cacheLanguageFrench is the French language selector.
	cacheLanguageFrench = "fr"

	// cacheLanguageGerman is the German language selector, the only variant that expires.
	cacheLanguageGerman = "de"

	// cacheOutcomeMiss is the action-cache header value for a handler invocation.
	cacheOutcomeMiss = "MISS"

	// cacheOutcomeHit is the action-cache header value for a cached response.
	cacheOutcomeHit = "HIT"

	// cacheKeyPrimary is the suffix of the repeatedly requested cache key.
	cacheKeyPrimary = "A"

	// cacheKeySecondary is the suffix of the cache key requested once.
	cacheKeySecondary = "B"

	// cacheFrenchExecution is the handler invocation that serves the French variant.
	cacheFrenchExecution = 3

	// cacheFirstGermanExecution is the handler invocation that first serves German.
	cacheFirstGermanExecution = 4

	// cacheSecondGermanExecution is the handler invocation after German expires.
	cacheSecondGermanExecution = 5

	// cacheGermanExpiryStep is the request step that waits for the German variant to expire.
	cacheGermanExpiryStep = 7

	// cacheGermanExpiryWait outlasts the German variant's 100 ms lifetime.
	cacheGermanExpiryWait = 200 * time.Millisecond
)

// CacheRead records the handler invocation ID from the typed response and the actual
// X-Action-Cache header from the Piko action dispatcher.
type CacheRead struct {
	// Key is the action-cache input key.
	Key string `json:"key"`

	// Language is the request's language selector.
	Language string `json:"language"`

	// Outcome is the observed action-cache response header.
	Outcome string `json:"outcome"`

	// Execution identifies the handler invocation.
	Execution int64 `json:"execution"`
}

// expectedCacheReads models action-cache hits across keys and languages.
//
// Takes label (string) which is the generated action label.
//
// Returns []CacheRead which lists cache read values in their recorded order.
func expectedCacheReads(label string) []CacheRead {
	first, second := label+"/"+cacheKeyPrimary, label+"/"+cacheKeySecondary
	return []CacheRead{
		{Key: first, Language: cacheLanguageEnglish, Execution: 1, Outcome: cacheOutcomeMiss},
		{Key: first, Language: cacheLanguageEnglish, Execution: 1, Outcome: cacheOutcomeHit},
		{Key: second, Language: cacheLanguageEnglish, Execution: 2, Outcome: cacheOutcomeMiss},
		{Key: first, Language: cacheLanguageEnglish, Execution: 1, Outcome: cacheOutcomeHit},
		{Key: first, Language: cacheLanguageFrench, Execution: cacheFrenchExecution, Outcome: cacheOutcomeMiss},
		{Key: first, Language: cacheLanguageEnglish, Execution: 1, Outcome: cacheOutcomeHit},
		{Key: first, Language: cacheLanguageGerman, Execution: cacheFirstGermanExecution, Outcome: cacheOutcomeMiss},
		{Key: first, Language: cacheLanguageGerman, Execution: cacheSecondGermanExecution, Outcome: cacheOutcomeMiss},
		{Key: first, Language: cacheLanguageFrench, Execution: cacheFrenchExecution, Outcome: cacheOutcomeHit},
	}
}

// observeActionCache records action-cache behaviour across keys and languages.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes token (string) which is the action CSRF token.
// Takes ephemeral (string) which is the form's ephemeral CSRF token.
// Takes label (string) which is the generated action label.
//
// Returns []CacheRead which lists cache read values in their recorded order.
// Returns error which reports action-request, response-decoding or cache-header failure,
// or nil on success.
func observeActionCache(ctx context.Context, client *http.Client, base, token, ephemeral, label string) ([]CacheRead, error) {
	var reads []CacheRead
	for step, request := range []struct{ suffix, language string }{
		{suffix: cacheKeyPrimary, language: cacheLanguageEnglish},
		{suffix: cacheKeyPrimary, language: cacheLanguageEnglish},
		{suffix: cacheKeySecondary, language: cacheLanguageEnglish},
		{suffix: cacheKeyPrimary, language: cacheLanguageEnglish},
		{suffix: cacheKeyPrimary, language: cacheLanguageFrench},
		{suffix: cacheKeyPrimary, language: cacheLanguageEnglish},
		{suffix: cacheKeyPrimary, language: cacheLanguageGerman},
		{suffix: cacheKeyPrimary, language: cacheLanguageGerman},
		{suffix: cacheKeyPrimary, language: cacheLanguageFrench},
	} {
		if step == cacheGermanExpiryStep {
			select {
			case <-ctx.Done():
				return reads, context.Cause(ctx)
			case <-time.After(cacheGermanExpiryWait):
			}
		}
		value, err := fetchCachedAction(ctx, client, base, token, ephemeral, label+"/"+request.suffix, request.language)
		if err != nil {
			return reads, fmt.Errorf("action/cache-request: step %d: %w", step, err)
		}
		reads = append(reads, value)
	}
	return reads, nil
}

// fetchCachedAction submits one cacheable action and reads its cache outcome.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes token (string) which is the action CSRF token.
// Takes ephemeral (string) which is the form's ephemeral CSRF token.
// Takes key (string) which is the cache input key.
// Takes language (string) which is the request's language selector.
//
// Returns CacheRead which records the handler invocation ID and cache outcome.
// Returns error which reports action-request, response-decoding or cache-header failure,
// or nil on success.
func fetchCachedAction(ctx context.Context, client *http.Client, base, token, ephemeral, key, language string) (CacheRead, error) {
	var value CacheRead
	body, err := json.Marshal(map[string]string{"key": key, "_csrf_ephemeral_token": ephemeral})
	if err != nil {
		return value, fmt.Errorf("action/cache-encode: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/_piko/actions/fuzz.Snapshot", bytes.NewReader(body))
	if err != nil {
		return value, err
	}
	setActionHeaders(request, mediaJSON, mediaJSON, token)
	request.Header.Set("Accept-Language", language)
	response, err := client.Do(request)
	if err != nil {
		return value, err
	}
	if response.StatusCode != http.StatusOK {
		return value, errors.Join(fmt.Errorf("HTTP %d", response.StatusCode), closeResponse(response))
	}
	if err := decodeAction(response, &value); err != nil {
		return value, err
	}
	value.Outcome = response.Header.Get("X-Action-Cache")
	return value, nil
}
