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

package memory

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

func TestIsSameDay(t *testing.T) {
	t.Parallel()

	t.Run("same day different hours", func(t *testing.T) {
		t.Parallel()

		morning := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
		evening := time.Date(2026, 3, 15, 20, 30, 0, 0, time.UTC)

		assert.True(t, isSameDay(morning, evening))
	})

	t.Run("different days", func(t *testing.T) {
		t.Parallel()

		today := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
		tomorrow := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)

		assert.False(t, isSameDay(today, tomorrow))
	})

	t.Run("midnight boundary", func(t *testing.T) {
		t.Parallel()

		beforeMidnight := time.Date(2026, 3, 15, 23, 59, 0, 0, time.UTC)
		afterMidnight := time.Date(2026, 3, 16, 0, 1, 0, 0, time.UTC)

		assert.False(t, isSameDay(beforeMidnight, afterMidnight))
	})

	t.Run("different months same day of month", func(t *testing.T) {
		t.Parallel()

		march := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
		april := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)

		assert.False(t, isSameDay(march, april))
	})
}

func TestTruncateToDay(t *testing.T) {
	t.Parallel()

	input := time.Date(2026, 3, 15, 14, 35, 22, 123456789, time.UTC)
	result := truncateToDay(input)

	expected := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
}

func TestStore_Record(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	firstCost := &llm_dto.CostEstimate{
		TotalCost: maths.NewMoneyFromString("5.00", "USD"),
	}
	secondCost := &llm_dto.CostEstimate{
		TotalCost: maths.NewMoneyFromString("3.50", "USD"),
	}

	require.NoError(t, store.Record(ctx, "test-scope", firstCost))
	require.NoError(t, store.Record(ctx, "test-scope", secondCost))

	status, err := store.GetStatus(ctx, "test-scope")
	require.NoError(t, err)

	expectedTotal := maths.NewMoneyFromString("8.50", "USD")
	assert.True(t, status.TotalSpent.CheckEquals(expectedTotal))
	assert.True(t, status.HourlySpent.CheckEquals(expectedTotal))
	assert.True(t, status.DailySpent.CheckEquals(expectedTotal))
}

func TestStore_Record_HourlyReset(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	firstCost := &llm_dto.CostEstimate{
		TotalCost: maths.NewMoneyFromString("5.00", "USD"),
	}
	require.NoError(t, store.Record(ctx, "test-scope", firstCost))

	mockClock.Advance(61 * time.Minute)

	secondCost := &llm_dto.CostEstimate{
		TotalCost: maths.NewMoneyFromString("2.00", "USD"),
	}
	require.NoError(t, store.Record(ctx, "test-scope", secondCost))

	status, err := store.GetStatus(ctx, "test-scope")
	require.NoError(t, err)

	expectedTotal := maths.NewMoneyFromString("7.00", "USD")
	expectedHourly := maths.NewMoneyFromString("2.00", "USD")
	expectedDaily := maths.NewMoneyFromString("7.00", "USD")

	assert.True(t, status.TotalSpent.CheckEquals(expectedTotal))
	assert.True(t, status.HourlySpent.CheckEquals(expectedHourly))
	assert.True(t, status.DailySpent.CheckEquals(expectedDaily))
}

func TestStore_Record_DailyReset(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	firstCost := &llm_dto.CostEstimate{
		TotalCost: maths.NewMoneyFromString("5.00", "USD"),
	}
	require.NoError(t, store.Record(ctx, "test-scope", firstCost))

	mockClock.Advance(25 * time.Hour)

	secondCost := &llm_dto.CostEstimate{
		TotalCost: maths.NewMoneyFromString("3.00", "USD"),
	}
	require.NoError(t, store.Record(ctx, "test-scope", secondCost))

	status, err := store.GetStatus(ctx, "test-scope")
	require.NoError(t, err)

	expectedTotal := maths.NewMoneyFromString("8.00", "USD")
	expectedHourly := maths.NewMoneyFromString("3.00", "USD")
	expectedDaily := maths.NewMoneyFromString("3.00", "USD")

	assert.True(t, status.TotalSpent.CheckEquals(expectedTotal))
	assert.True(t, status.HourlySpent.CheckEquals(expectedHourly))
	assert.True(t, status.DailySpent.CheckEquals(expectedDaily))
}

func TestStore_GetStatus(t *testing.T) {
	t.Parallel()

	t.Run("non-existent scope", func(t *testing.T) {
		t.Parallel()

		startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
		mockClock := clock.NewMockClock(startTime)
		store := New(WithClock(mockClock))
		ctx := context.Background()

		status, err := store.GetStatus(ctx, "missing-scope")
		require.NoError(t, err)

		assert.Equal(t, "missing-scope", status.Scope)

		zeroMoney := maths.ZeroMoney(llm_dto.CostCurrency)
		assert.True(t, status.TotalSpent.CheckEquals(zeroMoney))
		assert.True(t, status.DailySpent.CheckEquals(zeroMoney))
		assert.True(t, status.HourlySpent.CheckEquals(zeroMoney))
		assert.Equal(t, int64(0), status.RequestCount)
		assert.Equal(t, int64(0), status.TokenCount)
	})

	t.Run("after records", func(t *testing.T) {
		t.Parallel()

		startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
		mockClock := clock.NewMockClock(startTime)
		store := New(WithClock(mockClock))
		ctx := context.Background()

		cost := &llm_dto.CostEstimate{
			TotalCost: maths.NewMoneyFromString("12.50", "USD"),
		}
		require.NoError(t, store.Record(ctx, "scope-a", cost))
		require.NoError(t, store.IncrementRequests(ctx, "scope-a", 3))
		require.NoError(t, store.IncrementTokens(ctx, "scope-a", 1500))

		status, err := store.GetStatus(ctx, "scope-a")
		require.NoError(t, err)

		assert.Equal(t, "scope-a", status.Scope)

		expectedSpent := maths.NewMoneyFromString("12.50", "USD")
		assert.True(t, status.TotalSpent.CheckEquals(expectedSpent))
		assert.True(t, status.DailySpent.CheckEquals(expectedSpent))
		assert.True(t, status.HourlySpent.CheckEquals(expectedSpent))
		assert.Equal(t, int64(3), status.RequestCount)
		assert.Equal(t, int64(1500), status.TokenCount)
	})
}

func TestStore_GetStatus_WindowAdjustment(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	cost := &llm_dto.CostEstimate{
		TotalCost: maths.NewMoneyFromString("7.00", "USD"),
	}
	require.NoError(t, store.Record(ctx, "test-scope", cost))

	mockClock.Advance(61 * time.Minute)

	status, err := store.GetStatus(ctx, "test-scope")
	require.NoError(t, err)

	expectedTotal := maths.NewMoneyFromString("7.00", "USD")
	zeroMoney := maths.ZeroMoney(llm_dto.CostCurrency)

	assert.True(t, status.TotalSpent.CheckEquals(expectedTotal))
	assert.True(t, status.HourlySpent.CheckEquals(zeroMoney))
}

func TestStore_CheckAndReserve_WithinLimits(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	limits := llm_dto.BudgetLimits{
		MaxTotalSpend:  maths.NewMoneyFromString("100.00", "USD"),
		MaxDailySpend:  maths.NewMoneyFromString("50.00", "USD"),
		MaxHourlySpend: maths.NewMoneyFromString("20.00", "USD"),
	}

	reserveAmount := maths.NewMoneyFromString("10.00", "USD")
	require.NoError(t, store.CheckAndReserve(ctx, "test-scope", reserveAmount, limits))

	status, err := store.GetStatus(ctx, "test-scope")
	require.NoError(t, err)

	assert.True(t, status.TotalSpent.CheckEquals(reserveAmount))
	assert.True(t, status.HourlySpent.CheckEquals(reserveAmount))
	assert.True(t, status.DailySpent.CheckEquals(reserveAmount))
}

func TestStore_CheckAndReserve_TotalExceeded(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	limits := llm_dto.BudgetLimits{
		MaxTotalSpend: maths.NewMoneyFromString("100.00", "USD"),
	}

	initialReserve := maths.NewMoneyFromString("90.00", "USD")
	require.NoError(t, store.CheckAndReserve(ctx, "test-scope", initialReserve, limits))

	overflowReserve := maths.NewMoneyFromString("20.00", "USD")
	err := store.CheckAndReserve(ctx, "test-scope", overflowReserve, limits)

	assert.ErrorIs(t, err, llm_domain.ErrBudgetExceeded)
}

func TestStore_CheckAndReserve_DailyExceeded(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	limits := llm_dto.BudgetLimits{
		MaxDailySpend: maths.NewMoneyFromString("100.00", "USD"),
	}

	initialReserve := maths.NewMoneyFromString("90.00", "USD")
	require.NoError(t, store.CheckAndReserve(ctx, "test-scope", initialReserve, limits))

	overflowReserve := maths.NewMoneyFromString("20.00", "USD")
	err := store.CheckAndReserve(ctx, "test-scope", overflowReserve, limits)

	assert.ErrorIs(t, err, llm_domain.ErrBudgetExceeded)
}

func TestStore_CheckAndReserve_HourlyExceeded(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	limits := llm_dto.BudgetLimits{
		MaxHourlySpend: maths.NewMoneyFromString("100.00", "USD"),
	}

	initialReserve := maths.NewMoneyFromString("90.00", "USD")
	require.NoError(t, store.CheckAndReserve(ctx, "test-scope", initialReserve, limits))

	overflowReserve := maths.NewMoneyFromString("20.00", "USD")
	err := store.CheckAndReserve(ctx, "test-scope", overflowReserve, limits)

	assert.ErrorIs(t, err, llm_domain.ErrBudgetExceeded)
}

func TestStore_CheckAndReserve_ZeroLimitMeansUnlimited(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	limits := llm_dto.BudgetLimits{
		MaxTotalSpend: maths.ZeroMoney(llm_dto.CostCurrency),
	}

	largeAmount := maths.NewMoneyFromString("999999.00", "USD")
	err := store.CheckAndReserve(ctx, "test-scope", largeAmount, limits)

	assert.NoError(t, err)
}

func TestStore_CheckAndReserve_WindowReset(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	limits := llm_dto.BudgetLimits{
		MaxHourlySpend: maths.NewMoneyFromString("100.00", "USD"),
	}

	nearLimit := maths.NewMoneyFromString("95.00", "USD")
	require.NoError(t, store.CheckAndReserve(ctx, "test-scope", nearLimit, limits))

	overflowReserve := maths.NewMoneyFromString("10.00", "USD")
	err := store.CheckAndReserve(ctx, "test-scope", overflowReserve, limits)
	require.ErrorIs(t, err, llm_domain.ErrBudgetExceeded)

	mockClock.Advance(61 * time.Minute)

	err = store.CheckAndReserve(ctx, "test-scope", overflowReserve, limits)
	assert.NoError(t, err)
}

func TestStore_Unreserve(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	cost := &llm_dto.CostEstimate{
		TotalCost: maths.NewMoneyFromString("10.00", "USD"),
	}
	require.NoError(t, store.Record(ctx, "test-scope", cost))

	unreserveAmount := maths.NewMoneyFromString("4.00", "USD")
	require.NoError(t, store.Unreserve(ctx, "test-scope", unreserveAmount))

	status, err := store.GetStatus(ctx, "test-scope")
	require.NoError(t, err)

	expectedRemaining := maths.NewMoneyFromString("6.00", "USD")
	assert.True(t, status.TotalSpent.CheckEquals(expectedRemaining))
	assert.True(t, status.HourlySpent.CheckEquals(expectedRemaining))
	assert.True(t, status.DailySpent.CheckEquals(expectedRemaining))

	err = store.Unreserve(ctx, "non-existent-scope", maths.NewMoneyFromString("1.00", "USD"))
	assert.NoError(t, err)
}

func TestStore_IncrementRequests(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	require.NoError(t, store.IncrementRequests(ctx, "test-scope", 5))
	require.NoError(t, store.IncrementRequests(ctx, "test-scope", 3))

	status, err := store.GetStatus(ctx, "test-scope")
	require.NoError(t, err)

	assert.Equal(t, int64(8), status.RequestCount)
}

func TestStore_IncrementTokens(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	require.NoError(t, store.IncrementTokens(ctx, "test-scope", 1000))
	require.NoError(t, store.IncrementTokens(ctx, "test-scope", 500))

	status, err := store.GetStatus(ctx, "test-scope")
	require.NoError(t, err)

	assert.Equal(t, int64(1500), status.TokenCount)
}

func TestStore_Reset(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	cost := &llm_dto.CostEstimate{
		TotalCost: maths.NewMoneyFromString("10.00", "USD"),
	}
	require.NoError(t, store.Record(ctx, "scope-a", cost))
	require.NoError(t, store.Record(ctx, "scope-b", cost))

	require.NoError(t, store.Reset(ctx, "scope-a"))

	statusA, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	zeroMoney := maths.ZeroMoney(llm_dto.CostCurrency)
	assert.True(t, statusA.TotalSpent.CheckEquals(zeroMoney))

	statusB, err := store.GetStatus(ctx, "scope-b")
	require.NoError(t, err)
	expectedSpent := maths.NewMoneyFromString("10.00", "USD")
	assert.True(t, statusB.TotalSpent.CheckEquals(expectedSpent))
}

func TestStore_ResetAll(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	cost := &llm_dto.CostEstimate{
		TotalCost: maths.NewMoneyFromString("10.00", "USD"),
	}
	require.NoError(t, store.Record(ctx, "scope-a", cost))
	require.NoError(t, store.Record(ctx, "scope-b", cost))

	require.NoError(t, store.ResetAll(ctx))

	zeroMoney := maths.ZeroMoney(llm_dto.CostCurrency)

	statusA, err := store.GetStatus(ctx, "scope-a")
	require.NoError(t, err)
	assert.True(t, statusA.TotalSpent.CheckEquals(zeroMoney))

	statusB, err := store.GetStatus(ctx, "scope-b")
	require.NoError(t, err)
	assert.True(t, statusB.TotalSpent.CheckEquals(zeroMoney))
}

func TestStore_ConcurrentCheckAndReserve(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)
	store := New(WithClock(mockClock))
	ctx := context.Background()

	limits := llm_dto.BudgetLimits{
		MaxHourlySpend: maths.NewMoneyFromString("100.00", "USD"),
	}

	reserveAmount := maths.NewMoneyFromString("10.00", "USD")
	goroutineCount := 20
	var successCount atomic.Int64
	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutineCount)
	for range goroutineCount {
		go func() {
			defer waitGroup.Done()
			if store.CheckAndReserve(ctx, "test-scope", reserveAmount, limits) == nil {
				successCount.Add(1)
			}
		}()
	}

	waitGroup.Wait()

	assert.Equal(t, int64(10), successCount.Load())

	status, err := store.GetStatus(ctx, "test-scope")
	require.NoError(t, err)

	expectedTotal := maths.NewMoneyFromString("100.00", "USD")
	assert.True(t, status.HourlySpent.CheckEquals(expectedTotal))
}
