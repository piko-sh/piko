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

package shimcontracts

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_adapters"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// rateHistoryLimit bounds the replayed choice bytes driving extra requests.
	rateHistoryLimit = 16

	// tokenRate is the token-bucket refill rate in tokens per second.
	tokenRate = 2

	// tokenInterval is the half-second virtual time of one token-bucket credit and of the
	// final opening advances.
	tokenInterval = 500 * time.Millisecond

	// drawnPause is the half-minute drawn virtual-time advance.
	drawnPause = 30 * time.Second

	// nearWindowEnd advances the opening sequence to just before the first window ends.
	nearWindowEnd = 58 * time.Second

	// openingBucketSteps is the length of the fixed token-bucket sequence.
	openingBucketSteps = 7

	// storePolicyLimit is the capacity of the counter-store policy checks.
	storePolicyLimit = 2
)

var (
	// rateNames names the two independently limited fixture actions.
	rateNames = [2]string{"notes.Save", "notes.Preview"}

	// rateClients identifies the two independently limited fixture clients.
	rateClients = [2]string{"192.0.2.11", "192.0.2.12"}

	// errRateOffline marks the controlled counter-store outage.
	errRateOffline = errors.New("fixture counter offline")
)

// rateAction supplies an action's configured rate limit.
type rateAction int

// RateLimit returns the fixture action's configured rate limit.
//
// Returns *daemon_domain.RateLimit which holds the fixture action's configured limit.
func (a rateAction) RateLimit() *daemon_domain.RateLimit {
	var limit daemon_domain.RateLimit
	limit.RequestsPerMinute = int(a)
	return &limit
}

// rateWindow models one fixed rate-limit window.
type rateWindow struct {
	// end is the modelled window expiry.
	end time.Time

	// count is the modelled request count in this window.
	count int
}

// rateBucketStep selects one token-bucket request in the fixed opening history.
type rateBucketStep struct {
	// key selects the bucket.
	key int

	// cost is the requested token count.
	cost int

	// ticks is the number of half-second advances before the request.
	ticks int

	// reset deletes the bucket before the request.
	reset bool
}

// tokenModel is the integer-credit model of two token buckets.
//
// One tick is half a second at two tokens per second, so the model counts integer
// credits. An untouched key also earns credits, up to capacity.
type tokenModel struct {
	// configuration is the production token-bucket configuration.
	configuration ratelimiter_dto.TokenBucketConfig

	// balances holds each bucket's modelled credits.
	balances [2]int

	// capacity is the token bucket capacity.
	capacity int
}

// rateOpeningStep selects one request of the fixed opening sequence.
type rateOpeningStep struct {
	// advance is the virtual time to pass before the request.
	advance time.Duration

	// action selects one of the two limited actions.
	action int

	// client selects one of the two modelled clients.
	client int
}

// rateScenario models limits for two actions and two clients.
type rateScenario struct {
	// router is the mounted production action router.
	router http.Handler

	// clock is the virtual clock shared by the service and model.
	clock *clock.MockClock

	// limiter is the production rate-limit service.
	limiter *ratelimiter_domain.Limiter

	// windows holds independent windows for each action and client.
	windows [2][2]rateWindow

	// limits holds the two action capacities.
	limits [2]int

	// invokes counts handler calls separately for each action.
	invokes [2]int

	// allowed counts admitted requests.
	allowed int64

	// denied counts refused requests.
	denied int64
}

// request runs one action request and updates its independent rate-window model.
//
// Takes action (int) which selects one of the two independently limited actions.
// Takes client (int) which selects one of the two independently modelled clients.
//
// Returns error which reports a response or rate-counter mismatch with the independent
// model, or nil on success.
func (s *rateScenario) request(ctx context.Context, action, client int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := s.clock.Now()
	limit := s.limits[action]
	window := &s.windows[action][client]
	if !now.Before(window.end) {
		window.end = now.Add(time.Minute)
		window.count = 0
	}
	window.count++
	before := s.invokes
	holder := new(daemon_dto.PikoRequestCtx)
	holder.ClientIP = rateClients[client]
	requestCtx := daemon_dto.WithPikoRequestCtx(ctx, holder)
	request := httptest.NewRequestWithContext(requestCtx, http.MethodPost, "/_piko/actions/"+rateNames[action], strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, request)
	if err := checkRateResponse(recorder, limit, *window, now); err != nil {
		return fmt.Errorf("%w (action %d client %d)", err, action, client)
	}
	if window.count <= limit {
		s.allowed++
		before[action]++
	} else {
		s.denied++
	}
	if s.invokes != before {
		return errors.New("rate-dispatch: wrong action invocation count")
	}
	return nil
}

// tokenHistory checks a token bucket against an integer-credit model.
//
// Takes capacity (int) which is the token bucket capacity.
// Takes history ([]byte) which selects token-bucket requests and virtual time advances.
//
// Returns error which reports bucket deletion, cancellation or an integer-credit model
// mismatch, or nil on success.
func (s *rateScenario) tokenHistory(ctx context.Context, capacity int, history []byte) error {
	model := &tokenModel{
		configuration: ratelimiter_dto.TokenBucketConfig{Rate: tokenRate, Burst: capacity},
		balances:      [2]int{capacity, capacity},
		capacity:      capacity,
	}
	steps := make([]rateBucketStep, 0, openingBucketSteps+len(history))
	steps = append(steps,
		rateBucketStep{key: 0, cost: capacity, ticks: 0, reset: false},
		rateBucketStep{key: 0, cost: 1, ticks: 0, reset: false},
		rateBucketStep{key: 1, cost: capacity, ticks: 0, reset: false},
		rateBucketStep{key: 0, cost: 2, ticks: 1, reset: false},
		rateBucketStep{key: 0, cost: 1, ticks: 0, reset: false},
		rateBucketStep{key: 1, cost: capacity, ticks: 0, reset: true},
		rateBucketStep{key: 0, cost: capacity, ticks: capacity + 1, reset: false},
	)
	for _, choice := range history {
		steps = append(steps, rateBucketStep{
			key:   int(choice & 1),
			cost:  1 + int(choice>>1)%capacity,
			ticks: int(choice >> 5 & 3),
			reset: choice&128 != 0,
		})
	}
	for _, step := range steps {
		if err := s.take(ctx, model, step); err != nil {
			return err
		}
	}
	return nil
}

// take advances virtual time, requests tokens and compares the decision with the model.
//
// Takes model (*tokenModel) which holds the modelled bucket credits.
// Takes step (rateBucketStep) which selects the bucket, cost, time advance and reset.
//
// Returns error which reports cancellation, deletion failure or a decision that differs
// from the model.
func (s *rateScenario) take(ctx context.Context, model *tokenModel, step rateBucketStep) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.clock.Advance(time.Duration(step.ticks) * tokenInterval)
	for index := range model.balances {
		model.balances[index] = min(model.capacity, model.balances[index]+step.ticks)
	}
	if step.reset {
		if err := s.limiter.DeleteBucket(ctx, rateNames[step.key]); err != nil {
			return fmt.Errorf("rate-bucket-delete: store deletion failed: %w", err)
		}
		model.balances[step.key] = model.capacity
	}
	allowed := model.balances[step.key] >= step.cost
	err := s.limiter.AllowTokenBucket(ctx, rateNames[step.key], float64(step.cost), model.configuration)
	if !allowed {
		s.denied++
		if !errors.Is(err, ratelimiter_domain.ErrRateLimited) {
			return errors.New("rate-bucket-deny: depleted credits did not yield the rate-limited error")
		}
		return nil
	}
	model.balances[step.key] -= step.cost
	s.allowed++
	if err != nil {
		return errors.New("rate-bucket-allow: available credits refused")
	}
	return nil
}

// opening runs the fixed request sequence across both windows and the window boundary.
//
// The other action and client start offset from the first window. Half a second before
// the window ends, a refusal must still ask the client to wait at least one second rather
// than retry immediately.
//
// Returns error which reports a response or counter mismatch.
func (s *rateScenario) opening(ctx context.Context) error {
	for range s.limits[0] + 1 {
		if err := s.request(ctx, 0, 0); err != nil {
			return err
		}
	}
	steps := []rateOpeningStep{
		{advance: time.Second, action: 1, client: 0},
		{advance: 0, action: 0, client: 1},
		{advance: nearWindowEnd, action: 0, client: 0},
		{advance: tokenInterval, action: 0, client: 0},
		{advance: tokenInterval, action: 0, client: 0},
		{advance: 0, action: 1, client: 0},
	}
	for _, step := range steps {
		if step.advance > 0 {
			s.clock.Advance(step.advance)
		}
		if err := s.request(ctx, step.action, step.client); err != nil {
			return err
		}
	}
	return nil
}

// rateOffline simulates an unavailable rate-limit counter.
type rateOffline struct {
	// CounterStorePort is the delegated healthy counter store.
	ratelimiter_domain.CounterStorePort

	// offline selects the counter-store failure.
	offline bool
}

// IncrementAndGet returns the fixture counter result or its controlled outage.
//
// Takes key (string) which is the counter-store bucket key.
// Takes delta (int64) which is the requested counter increment.
// Takes window (time.Duration) which is the rate-limit window duration.
//
// Returns ratelimiter_dto.CounterResult which holds the delegated counter-store result
// while the fixture is online.
// Returns error which reports the controlled outage or underlying counter-store failure,
// or nil on success.
func (s *rateOffline) IncrementAndGet(ctx context.Context, key string, delta int64, window time.Duration) (ratelimiter_dto.CounterResult, error) {
	if s.offline {
		return ratelimiter_dto.CounterResult{}, errRateOffline
	}
	return s.CounterStorePort.IncrementAndGet(ctx, key, delta, window)
}

// RunRateLimit checks mounted action responses against a model with independent windows
// for two actions and two clients.
//
// Takes data ([]byte) which holds the replayable choice bytes.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func RunRateLimit(ctx context.Context, data []byte) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	var config byte
	if len(data) > 0 {
		config, data = data[0], data[1:]
	}
	mockClock := clock.NewMockClock(time.Unix(1700000000, 0).UTC())
	var counterConfig ratelimiter_adapters.CacheCounterStoreConfig
	counterConfig.Clock = mockClock
	counterConfig.CacheService = cache_domain.NewService("")
	counterConfig.MaximumEntries = rateHistoryLimit
	counters, err := ratelimiter_adapters.NewCacheCounterStore(ctx, counterConfig)
	if err != nil {
		return fmt.Errorf("rate-setup: local cache: %w", err)
	}
	defer func() {
		if closeErr := counters.Close(context.WithoutCancel(ctx)); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("rate-setup: close local cache: %w", closeErr))
		}
	}()
	buckets := ratelimiter_adapters.NewInMemoryTokenBucketStore(ratelimiter_adapters.WithInMemoryClock(mockClock))
	limiter := ratelimiter_domain.NewLimiter(buckets, counters, ratelimiter_domain.WithClock(mockClock), ratelimiter_domain.WithFailPolicy(ratelimiter_dto.FailClosed))
	service := security_domain.NewRateLimitService(limiter)
	scenario := newRateScenario(mockClock, limiter, service, config)
	if err := scenario.opening(ctx); err != nil {
		return err
	}
	history := data[:min(len(data), rateHistoryLimit)]
	advances := [...]time.Duration{0, time.Second, drawnPause, time.Minute}
	for _, choice := range history {
		mockClock.Advance(advances[choice>>2&3])
		if err := scenario.request(ctx, int(choice&1), int(choice>>1&1)); err != nil {
			return err
		}
	}
	if err := checkZeroCapacity(ctx, service, mockClock); err != nil {
		return err
	}
	scenario.denied++
	if err := scenario.tokenHistory(ctx, 2+int(config>>4&3), history); err != nil {
		return err
	}
	status, err := limiter.GetStatus(ctx)
	if err != nil || status.TotalAllowed != scenario.allowed || status.TotalDenied != scenario.denied || status.TotalErrors != 0 ||
		status.TotalChecks != scenario.allowed+scenario.denied {
		return errors.New("rate-accounting: inspector totals differ from independently observed outcomes")
	}
	return rateStorePolicies(ctx, mockClock, counters)
}

// newRateScenario mounts the two limited actions behind the production action handler.
//
// Takes mockClock (*clock.MockClock) which is the virtual clock.
// Takes limiter (*ratelimiter_domain.Limiter) which is the production limiter.
// Takes service (security_domain.RateLimitService) which is the production rate-limit
// service.
// Takes config (byte) which selects the two action capacities.
//
// Returns *rateScenario which owns the router and the independent model.
func newRateScenario(mockClock *clock.MockClock, limiter *ratelimiter_domain.Limiter, service security_domain.RateLimitService, config byte) *rateScenario {
	scenario := &rateScenario{
		router:  nil,
		clock:   mockClock,
		limiter: limiter,
		windows: [2][2]rateWindow{},
		limits:  [2]int{1 + int(config%4), 1 + int((config>>2)%4)},
		invokes: [2]int{},
		allowed: 0,
		denied:  0,
	}
	var values security_dto.RateLimitValues
	values.Enabled = true
	values.HeadersEnabled = true
	handler := daemon_adapters.NewActionHandler(nil, actionBodyLimit, service, values, false, nil, nil)
	for index, name := range rateNames {
		var entry daemon_adapters.ActionHandlerEntry
		entry.Name = name
		entry.Method = http.MethodPost
		entry.Create = func() any { return rateAction(scenario.limits[index]) }
		entry.Invoke = func(context.Context, any, map[string]any) (any, error) {
			scenario.invokes[index]++
			return name, nil
		}
		handler.Register(entry)
	}
	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")
	scenario.router = router
	return scenario
}

// checkZeroCapacity checks that a configured capacity of zero, which is valid on the
// fixed-window service, refuses immediately for the full window.
//
// Takes service (security_domain.RateLimitService) which is the production rate-limit
// service.
// Takes mockClock (*clock.MockClock) which is the virtual clock.
//
// Returns error which reports a zero-capacity result that differs from the model.
func checkZeroCapacity(ctx context.Context, service security_domain.RateLimitService, mockClock *clock.MockClock) error {
	zero, err := service.CheckLimit(ctx, "maintenance", 0, time.Minute)
	if err != nil || zero.Allowed || zero.Remaining != 0 || zero.Limit != 0 || !zero.ResetAt.Equal(mockClock.Now().Add(time.Minute)) ||
		zero.RetryAfter != time.Minute {
		return errors.New("rate-zero: zero capacity did not refuse immediately for the full window")
	}
	return nil
}

// checkRateResponse checks the public response and rate-limit headers.
//
// Takes recorder (*httptest.ResponseRecorder) which holds the recorded HTTP response.
// Takes limit (int) which is the permitted action request count per minute.
// Takes window (rateWindow) which holds the independently modelled rate window.
// Takes now (time.Time) which is the independent model's current time.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkRateResponse(recorder *httptest.ResponseRecorder, limit int, window rateWindow, now time.Time) error {
	allowed := window.count <= limit
	wantStatus := http.StatusOK
	retry := ""
	if !allowed {
		wantStatus = http.StatusTooManyRequests
		retry = strconv.Itoa(max(int((window.end.Sub(now)+time.Second-1)/time.Second), 1))
	}
	if recorder.Code != wantStatus {
		return fmt.Errorf("rate-decision: HTTP %d, expected %d", recorder.Code, wantStatus)
	}
	header := recorder.Header()
	if header.Get("X-RateLimit-Limit") != strconv.Itoa(limit) || header.Get("X-RateLimit-Remaining") != strconv.Itoa(max(limit-window.count, 0)) {
		return errors.New("rate-quota: limit or remaining header differs")
	}
	if header.Get("X-RateLimit-Reset") != strconv.FormatInt(window.end.Unix(), 10) || header.Get("Retry-After") != retry {
		return errors.New("rate-reset: reset or retry header differs")
	}
	return nil
}

// rateStorePolicies checks fail-open and fail-closed counter-store policies.
//
// Takes mockClock (*clock.MockClock) which is the virtual clock.
// Takes counter (ratelimiter_domain.CounterStorePort) which is the controlled counter
// store.
//
// Returns error which reports cancellation or fallback, recovery or counter-accounting
// mismatch, or nil on success.
func rateStorePolicies(ctx context.Context, mockClock *clock.MockClock, counter ratelimiter_domain.CounterStorePort) error {
	for _, policy := range []ratelimiter_dto.FailPolicy{ratelimiter_dto.FailClosed, ratelimiter_dto.FailOpen} {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := rateStorePolicy(ctx, mockClock, counter, policy); err != nil {
			return err
		}
	}
	return nil
}

// rateStorePolicy checks one counter-store failure policy through an outage and its
// recovery.
//
// Takes mockClock (*clock.MockClock) which is the virtual clock.
// Takes counter (ratelimiter_domain.CounterStorePort) which is the controlled counter
// store.
// Takes policy (ratelimiter_dto.FailPolicy) which selects fail-open or fail-closed.
//
// Returns error which reports a fallback, recovery or counter-accounting mismatch.
func rateStorePolicy(ctx context.Context, mockClock *clock.MockClock, counter ratelimiter_domain.CounterStorePort, policy ratelimiter_dto.FailPolicy) error {
	store := &rateOffline{CounterStorePort: counter, offline: true}
	limiter := ratelimiter_domain.NewLimiter(nil, store, ratelimiter_domain.WithClock(mockClock), ratelimiter_domain.WithFailPolicy(policy))
	service := security_domain.NewRateLimitService(limiter)
	key := "status-" + strconv.Itoa(int(policy))
	result, err := service.CheckLimit(ctx, key, storePolicyLimit, time.Minute)
	var denied int64
	if policy == ratelimiter_dto.FailClosed {
		denied = 1
		if result.Allowed || !errors.Is(err, ratelimiter_domain.ErrStoreFailure) || !errors.Is(err, errRateOffline) {
			return errors.New("rate-store-closed: expected a refused result with the backend error")
		}
	} else if err != nil || !result.Allowed || result.Limit != storePolicyLimit || result.Remaining != 1 || result.RetryAfter != 0 ||
		!result.ResetAt.Equal(mockClock.Now().Add(time.Minute)) {
		return errors.New("rate-store-open: expected the documented provisional allowance")
	}
	store.offline = false
	result, err = service.CheckLimit(ctx, key, storePolicyLimit, time.Minute)
	if err != nil || !result.Allowed || result.Remaining != 1 {
		return errors.New("rate-store-recovery: failed operation consumed capacity")
	}
	status, err := limiter.GetStatus(ctx)
	if err != nil || status.TotalErrors != 1 || status.TotalChecks != 2 || status.TotalDenied != denied || status.TotalAllowed != 2-denied {
		return errors.New("rate-store-status: fallback and recovery counters differ")
	}
	return nil
}
