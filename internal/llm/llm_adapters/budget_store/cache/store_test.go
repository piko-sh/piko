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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

func newTestStore(t *testing.T) (*Store, *clock.MockClock) {
	t.Helper()

	mockClock := clock.NewMockClock(time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC))
	cacheService := cache_domain.NewService("otter")

	store, err := New(context.Background(), Config{
		CacheService: cacheService,
		Clock:        mockClock,
		Namespace:    "test:budget",
		MaximumSize:  1000,
	})
	require.NoError(t, err)

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
	ctx := context.Background()

	err := store.Record(ctx, "scope-a", costEstimate("1.50"))
	require.NoError(t, err)

	status, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, status.TotalSpent.CheckGreaterThan(maths.ZeroMoney(llm_dto.CostCurrency)))
	assert.True(t, status.TotalSpent.CheckEquals(money("1.50")))
}

func TestStore_Record_AccumulatesCost(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

	err := store.Unreserve(ctx, "nonexistent-scope", money("1.00"))
	require.NoError(t, err)
}

func TestStore_IncrementRequests(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
