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

package conformance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
)

type recordingT struct {
	failures []string
	mu       sync.Mutex
}

func (r *recordingT) Errorf(format string, arguments ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failures = append(r.failures, fmt.Sprintf(format, arguments...))
}

func (*recordingT) FailNow() { runtime.Goexit() }

func (*recordingT) Helper() {}

func (*recordingT) Context() context.Context { return context.Background() }

func (r *recordingT) failed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.failures) > 0
}

type earlyExpiringCache struct {
	cache_domain.ProviderPort[string, string]
	written map[string]time.Time
	ttls    map[string]time.Duration
	mu      sync.Mutex
}

func (c *earlyExpiringCache) SetWithTTL(_ context.Context, key string, _ string, ttl time.Duration, _ ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.written[key] = time.Now()
	c.ttls[key] = ttl
	return nil
}

func (c *earlyExpiringCache) GetIfPresent(_ context.Context, key string) (string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	written, ok := c.written[key]
	if !ok {
		return "", false, nil
	}
	return perEntryValue, time.Since(written) < c.ttls[key]/10, nil
}

func (*earlyExpiringCache) ComputeWithTTL(_ context.Context, _ string, computeFunction func(string, bool) cache_dto.ComputeResult[string]) (string, bool, error) {
	result := computeFunction("", false)
	return result.Value, true, nil
}

func runScenario(cache cache_domain.ProviderPort[string, string], check func(r perEntryRun)) bool {
	recorder := &recordingT{}
	run := perEntryRun{
		harness:      PerEntryHarness{Cache: cache, AdvanceTime: nil, Clock: nil},
		t:            recorder,
		ttl:          realTTL,
		subSecondTTL: realSubSecondTTL,
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		check(run)
	}()
	<-done
	return recorder.failed()
}

func TestRequireExpiresInRealTimeCatchesEarlyExpiry(t *testing.T) {
	t.Parallel()

	cache := &earlyExpiringCache{written: make(map[string]time.Time), ttls: make(map[string]time.Duration)}

	failed := runScenario(cache, func(r perEntryRun) {
		r.setWithTTL(perEntryKey, r.ttl)
		r.requireExpires(perEntryKey, r.ttl)
	})

	assert.True(t, failed, "an entry gone well before half its TTL must fail the expiry check")
}

func TestFixedWindowInRealTimeCatchesImmediateExpiry(t *testing.T) {
	t.Parallel()

	cache := &earlyExpiringCache{written: make(map[string]time.Time), ttls: make(map[string]time.Duration)}

	failed := runScenario(cache, testFixedWindowCounter)

	assert.True(t, failed, "a counter that resets on every increment must fail the fixed window check")
}
