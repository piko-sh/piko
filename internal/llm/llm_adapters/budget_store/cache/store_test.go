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

package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_adapters/provider_mock"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

func newTestStore(t *testing.T) (*Store, *clock.MockClock) {
	t.Helper()

	mockClock := clock.NewMockClock(time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC))
	cacheService := cache_domain.NewService("otter")

	store, err := New(t.Context(), Config{
		CacheService:   cacheService,
		Clock:          mockClock,
		Provider:       "",
		Namespace:      "test:budget",
		MaximumEntries: 1000,
	})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.Close(context.Background())) })

	return store, mockClock
}

func money(amount string) maths.Money {
	return maths.NewMoneyFromString(amount, llm_dto.CostCurrency)
}

func costEstimate(amount string) *llm_dto.CostEstimate {
	return &llm_dto.CostEstimate{
		TotalCost: money(amount),
	}
}

func TestStore_Record_CreatesEntry(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := t.Context()

	err := store.Record(ctx, "scope-a", costEstimate("1.50"))
	require.NoError(t, err)

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.TotalSpent.CheckGreaterThan(maths.ZeroMoney(llm_dto.CostCurrency)))
	assert.True(t, status.TotalSpent.CheckEquals(money("1.50")))
}

func TestStore_Record_AccumulatesCost(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := t.Context()

	err := store.Record(ctx, "scope-a", costEstimate("1.00"))
	require.NoError(t, err)

	err = store.Record(ctx, "scope-a", costEstimate("2.50"))
	require.NoError(t, err)

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.TotalSpent.CheckEquals(money("3.50")))
}

func TestStore_CheckAndReserve_WithinLimits(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := t.Context()

	limits := llm_dto.BudgetLimits{
		MaxTotalSpend: money("10.00"),
	}

	err := store.CheckAndReserve(ctx, "scope-a", money("3.00"), limits)
	require.NoError(t, err)

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.TotalSpent.CheckEquals(money("3.00")))
}

func TestStore_CheckAndReserve_ExceedsLimit(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := t.Context()

	limits := llm_dto.BudgetLimits{
		MaxTotalSpend: money("5.00"),
	}

	err := store.CheckAndReserve(ctx, "scope-a", money("3.00"), limits)
	require.NoError(t, err)

	err = store.CheckAndReserve(ctx, "scope-a", money("3.00"), limits)
	require.ErrorIs(t, err, llm_domain.ErrBudgetExceeded)

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.TotalSpent.CheckEquals(money("3.00")))
}

func TestStore_Unreserve_SubtractsCost(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := t.Context()

	limits := llm_dto.BudgetLimits{
		MaxTotalSpend: money("10.00"),
	}

	err := store.CheckAndReserve(ctx, "scope-a", money("5.00"), limits)
	require.NoError(t, err)

	err = store.Unreserve(ctx, "scope-a", money("2.00"))
	require.NoError(t, err)

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.TotalSpent.CheckEquals(money("3.00")))
}

func TestStore_Unreserve_MissingScope(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := t.Context()

	err := store.Unreserve(ctx, "nonexistent-scope", money("1.00"))
	require.NoError(t, err)
}

func TestStore_IncrementRequests(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := t.Context()

	err := store.IncrementRequests(ctx, "scope-a", 3)
	require.NoError(t, err)

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.Equal(t, int64(3), status.RequestCount)

	err = store.IncrementRequests(ctx, "scope-a", 2)
	require.NoError(t, err)

	status, err = store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.Equal(t, int64(5), status.RequestCount)
}

func TestStore_IncrementTokens(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := t.Context()

	err := store.IncrementTokens(ctx, "scope-a", 100)
	require.NoError(t, err)

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.Equal(t, int64(100), status.TokenCount)

	err = store.IncrementTokens(ctx, "scope-a", 50)
	require.NoError(t, err)

	status, err = store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.Equal(t, int64(150), status.TokenCount)
}

func TestStore_Reset_ClearsScope(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := t.Context()

	err := store.Record(ctx, "scope-a", costEstimate("5.00"))
	require.NoError(t, err)

	err = store.IncrementRequests(ctx, "scope-a", 10)
	require.NoError(t, err)

	err = store.IncrementTokens(ctx, "scope-a", 500)
	require.NoError(t, err)

	err = store.Reset(ctx, "scope-a")
	require.NoError(t, err)

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.TotalSpent.CheckEquals(maths.ZeroMoney(llm_dto.CostCurrency)))
	assert.True(t, status.HourlySpent.CheckEquals(maths.ZeroMoney(llm_dto.CostCurrency)))
	assert.True(t, status.DailySpent.CheckEquals(maths.ZeroMoney(llm_dto.CostCurrency)))
	assert.Equal(t, int64(0), status.RequestCount)
	assert.Equal(t, int64(0), status.TokenCount)
}

func TestStore_GetStatus_EmptyScope(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := t.Context()

	status, err := store.GetStatus(ctx, "never-used-scope")
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "never-used-scope", status.Scope)
	assert.True(t, status.TotalSpent.CheckEquals(maths.ZeroMoney(llm_dto.CostCurrency)))
	assert.True(t, status.DailySpent.CheckEquals(maths.ZeroMoney(llm_dto.CostCurrency)))
	assert.True(t, status.HourlySpent.CheckEquals(maths.ZeroMoney(llm_dto.CostCurrency)))
	assert.Equal(t, int64(0), status.RequestCount)
	assert.Equal(t, int64(0), status.TokenCount)
}

func TestStore_WindowReset_HourlyBoundary(t *testing.T) {
	store, mockClock := newTestStore(t)
	ctx := t.Context()

	err := store.Record(ctx, "scope-a", costEstimate("2.00"))
	require.NoError(t, err)

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.HourlySpent.CheckEquals(money("2.00")))
	assert.True(t, status.TotalSpent.CheckEquals(money("2.00")))

	mockClock.Advance(61 * time.Minute)

	err = store.Record(ctx, "scope-a", costEstimate("1.00"))
	require.NoError(t, err)

	status, err = store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.HourlySpent.CheckEquals(money("1.00")))
	assert.True(t, status.TotalSpent.CheckEquals(money("3.00")))
}

func TestStore_WindowReset_DailyBoundary(t *testing.T) {
	store, mockClock := newTestStore(t)
	ctx := t.Context()

	err := store.Record(ctx, "scope-a", costEstimate("4.00"))
	require.NoError(t, err)

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.DailySpent.CheckEquals(money("4.00")))
	assert.True(t, status.TotalSpent.CheckEquals(money("4.00")))

	mockClock.Advance(24 * time.Hour)

	err = store.Record(ctx, "scope-a", costEstimate("1.50"))
	require.NoError(t, err)

	status, err = store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.DailySpent.CheckEquals(money("1.50")))
	assert.True(t, status.TotalSpent.CheckEquals(money("5.50")))
}

func TestStore_Close_PropagatesCacheError(t *testing.T) {
	t.Parallel()

	closeErr := errors.New("close failed")
	store := &Store{cache: &retryingBudgetCache{closeErr: closeErr}, clock: clock.RealClock()}

	require.ErrorIs(t, store.Close(t.Context()), closeErr)
}

func TestStore_ConcurrentUpdatesAndReads(t *testing.T) {
	t.Parallel()

	const writers = 4
	const updatesPerWriter = 50

	store, _ := newTestStore(t)
	ctx := t.Context()
	require.NoError(t, store.Record(ctx, "scope-a", costEstimate("0.01")))

	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() {
			for range updatesPerWriter {
				assert.NoError(t, store.Record(ctx, "scope-a", costEstimate("0.01")))
				assert.NoError(t, store.IncrementRequests(ctx, "scope-a", 1))
				assert.NoError(t, store.Unreserve(ctx, "scope-a", money("0.00")))
			}
		})
		wg.Go(func() {
			for range updatesPerWriter {
				_, err := store.GetStatus(ctx, "scope-a")
				assert.NoError(t, err)
			}
		})
	}
	wg.Wait()

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.TotalSpent.CheckEquals(money("2.01")))
	assert.Equal(t, int64(writers*updatesPerWriter), status.RequestCount)
}

func TestStore_CheckAndReserve_RetriedCallbackReportsLastAttempt(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)
	nearLimit := getOrCreate(nil, false, now)
	nearLimit.TotalSpent = money("9.50")
	store := &Store{
		cache: &retryingBudgetCache{attempts: []*llm_dto.BudgetData{nearLimit, getOrCreate(nil, false, now)}},
		clock: clock.NewMockClock(now),
	}
	limits := llm_dto.BudgetLimits{MaxTotalSpend: money("10.00")}

	err := store.CheckAndReserve(t.Context(), "scope-a", money("1.00"), limits)

	require.NoError(t, err, "the result must come from the attempt the cache kept, not an earlier one")
}

func TestStore_CheckAndReserve_DenyWritesOnlyWhenAWindowResets(t *testing.T) {
	t.Parallel()

	store, mockClock := newTestStore(t)
	recorder := &actionRecordingCache{Cache: store.cache}
	store.cache = recorder
	ctx := t.Context()
	limits := llm_dto.BudgetLimits{MaxHourlySpend: money("1.00")}

	require.NoError(t, store.CheckAndReserve(ctx, "scope-a", money("1.00"), limits))
	require.ErrorIs(t, store.CheckAndReserve(ctx, "scope-a", money("0.50"), limits), llm_domain.ErrBudgetExceeded)
	require.ErrorIs(t, store.CheckAndReserve(ctx, "scope-b", money("2.00"), limits), llm_domain.ErrBudgetExceeded)

	assert.Equal(t, []cache_dto.ComputeAction{
		cache_dto.ComputeActionSet,
		cache_dto.ComputeActionNoop,
		cache_dto.ComputeActionNoop,
	}, recorder.actions, "a denied request that changes nothing must not write")

	mockClock.Advance(time.Hour)
	recorder.actions = nil
	require.ErrorIs(t, store.CheckAndReserve(ctx, "scope-a", money("2.00"), limits), llm_domain.ErrBudgetExceeded)
	assert.Equal(t, []cache_dto.ComputeAction{cache_dto.ComputeActionSet}, recorder.actions,
		"a denied request still stores a window reset")

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.HourlySpent.CheckIsZero())
	assert.True(t, status.TotalSpent.CheckEquals(money("1.00")))
}

func TestStore_UseAfterCloseDoesNotPanic(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	store, err := New(ctx, Config{CacheService: cache_domain.NewService("otter")})
	require.NoError(t, err)
	require.NoError(t, store.Close(ctx))

	limits := llm_dto.BudgetLimits{MaxTotalSpend: money("10.00")}
	assert.NotPanics(t, func() {
		_ = store.Record(ctx, "scope-a", costEstimate("1.00"))
		_ = store.CheckAndReserve(ctx, "scope-a", money("1.00"), limits)
		_ = store.Unreserve(ctx, "scope-a", money("1.00"))
		_, _ = store.GetStatus(ctx, "scope-a")
		_ = store.IncrementRequests(ctx, "scope-a", 1)
		_ = store.IncrementTokens(ctx, "scope-a", 1)
		_ = store.Reset(ctx, "scope-a")
		_ = store.Close(ctx)
	}, "a request still running when the store closes must not crash the process")
}

func TestStore_UsesConfiguredProvider(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	provider := provider_mock.NewMockProvider()
	service := cache_domain.NewService("shared")
	require.NoError(t, service.RegisterProvider(ctx, "shared", provider))
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	store, err := New(ctx, Config{
		CacheService:   service,
		Clock:          clock.NewMockClock(time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)),
		Provider:       "shared",
		Namespace:      "",
		MaximumEntries: 0,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close(context.Background()) })

	require.NoError(t, store.Record(ctx, "scope-a", costEstimate("1.50")))
	require.NoError(t, store.Record(ctx, "scope-a", costEstimate("0.50")))

	cacheAny, err := provider.CreateNamespaceTyped(DefaultNamespace, cache_dto.Options[string, []byte]{})
	require.NoError(t, err)
	byteCache, ok := cacheAny.(cache_domain.Cache[string, []byte])
	require.True(t, ok)
	_, found, err := byteCache.GetIfPresent(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, found, "the named provider holds the budget")

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.TotalSpent.CheckEquals(money("2.00")))
}

func TestNew_RemoteProviderUsesPerEntryExpiry(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	provider := &optionsRecordingProvider{MockProvider: provider_mock.NewMockProvider()}
	service := cache_domain.NewService("remote")
	require.NoError(t, service.RegisterProvider(ctx, "remote", provider))
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	store, err := New(ctx, Config{CacheService: service, Provider: "remote"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close(context.Background()) })

	options, ok := provider.options.(cache_dto.Options[string, []byte])
	require.True(t, ok, "the remote cache stores encoded bytes, got %T", provider.options)
	assert.True(t, cache_domain.IsPerEntryExpiry(options.ExpiryCalculator),
		"budgets on a remote provider must not take the provider's default TTL")
}

func TestNew_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		config Config
	}{
		{name: "missing cache service", config: Config{}},
		{name: "unknown provider", config: Config{CacheService: cache_domain.NewService(""), Provider: "missing"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			store, err := New(t.Context(), testCase.config)

			assert.Nil(t, store)
			require.Error(t, err)
		})
	}
}

type retryingBudgetCache struct {
	cache_domain.Cache[string, *llm_dto.BudgetData]
	closeErr error
	attempts []*llm_dto.BudgetData
}

func (c *retryingBudgetCache) Compute(
	_ context.Context,
	_ string,
	computeFunction func(oldValue *llm_dto.BudgetData, found bool) (*llm_dto.BudgetData, cache_dto.ComputeAction),
) (*llm_dto.BudgetData, bool, error) {
	var value *llm_dto.BudgetData
	for _, data := range c.attempts {
		value, _ = computeFunction(data, true)
	}
	return value, true, nil
}

func (c *retryingBudgetCache) Close(context.Context) error {
	return c.closeErr
}

type optionsRecordingProvider struct {
	*provider_mock.MockProvider
	options any
}

func (p *optionsRecordingProvider) CreateNamespaceTyped(namespace string, options any) (any, error) {
	p.options = options
	return p.MockProvider.CreateNamespaceTyped(namespace, options)
}

type actionRecordingCache struct {
	cache_domain.Cache[string, *llm_dto.BudgetData]
	actions []cache_dto.ComputeAction
}

func (c *actionRecordingCache) Compute(
	ctx context.Context,
	key string,
	computeFunction func(oldValue *llm_dto.BudgetData, found bool) (*llm_dto.BudgetData, cache_dto.ComputeAction),
) (*llm_dto.BudgetData, bool, error) {
	return c.Cache.Compute(ctx, key, func(oldValue *llm_dto.BudgetData, found bool) (*llm_dto.BudgetData, cache_dto.ComputeAction) {
		value, action := computeFunction(oldValue, found)
		c.actions = append(c.actions, action)
		return value, action
	})
}
