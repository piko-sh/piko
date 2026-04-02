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
	"fmt"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

const (
	// DefaultMaximumSize is the default maximum number of budget scope entries.
	DefaultMaximumSize = 10000

	// DefaultNamespace is the default cache namespace for budget data.
	DefaultNamespace = "llm:budget"
)

// Config configures the cache-backed budget store.
type Config struct {
	// CacheService is the cache service used for storing entries; must not be nil.
	CacheService cache_domain.Service

	// Clock provides time operations. If nil, defaults to RealClock.
	Clock clock.Clock

	// Namespace is the key prefix for all budget entries.
	Namespace string

	// MaximumSize is the maximum number of scope entries.
	MaximumSize int
}

// Store provides an LLM budget store using the internal cache service.
// It implements llm_domain.BudgetStorePort.
type Store struct {
	// cache stores budget data entries keyed by scope.
	cache cache_domain.Cache[string, *llm_dto.BudgetData]

	// clock provides the current time for window calculations.
	clock clock.Clock
}

var _ llm_domain.BudgetStorePort = (*Store)(nil)

// Record adds a cost entry for a scope.
//
// Takes scope (string) which identifies the budget scope.
// Takes cost (*llm_dto.CostEstimate) which is the cost to record.
//
// Returns error if the context is cancelled.
func (s *Store) Record(ctx context.Context, scope string, cost *llm_dto.CostEstimate) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if cost == nil {
		return nil
	}

	now := s.clock.Now()
	_, _, err := s.cache.Compute(ctx, scope, func(old *llm_dto.BudgetData, found bool) (*llm_dto.BudgetData, cache_dto.ComputeAction) {
		data := getOrCreate(old, found, now)
		resetStaleWindows(data, now)

		data.TotalSpent.AddInPlace(cost.TotalCost)
		data.HourlySpent.AddInPlace(cost.TotalCost)
		data.DailySpent.AddInPlace(cost.TotalCost)
		data.LastUpdated = now

		return data, cache_dto.ComputeActionSet
	})
	if err != nil {
		return fmt.Errorf("recording budget cost for scope %q: %w", scope, err)
	}

	return nil
}

// CheckAndReserve atomically checks whether the estimated cost fits within
// all configured limits and, if so, reserves it by adding the cost to the
// current spend counters.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes scope (string) which identifies the budget scope.
// Takes estimatedCost (maths.Money) which is the cost to reserve.
// Takes limits (llm_dto.BudgetLimits) which carries the spend limits.
//
// Returns error which is llm_domain.ErrBudgetExceeded if any limit would be
// breached.
func (s *Store) CheckAndReserve(ctx context.Context, scope string, estimatedCost maths.Money, limits llm_dto.BudgetLimits) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	now := s.clock.Now()
	var checkErr error

	_, _, err := s.cache.Compute(ctx, scope, func(old *llm_dto.BudgetData, found bool) (*llm_dto.BudgetData, cache_dto.ComputeAction) {
		data := getOrCreate(old, found, now)
		resetStaleWindows(data, now)

		if err := checkLimits(data, estimatedCost, limits); err != nil {
			checkErr = err
			return data, cache_dto.ComputeActionSet
		}

		data.TotalSpent.AddInPlace(estimatedCost)
		data.HourlySpent.AddInPlace(estimatedCost)
		data.DailySpent.AddInPlace(estimatedCost)
		data.LastUpdated = now

		return data, cache_dto.ComputeActionSet
	})
	if err != nil {
		return fmt.Errorf("checking and reserving budget for scope %q: %w", scope, err)
	}

	return checkErr
}

// Unreserve releases a previously reserved cost when the request fails before
// the actual usage is recorded.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes scope (string) which identifies the budget scope.
// Takes cost (maths.Money) which is the reserved cost to release.
//
// Returns error if the context is cancelled.
func (s *Store) Unreserve(ctx context.Context, scope string, cost maths.Money) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	_, _, err := s.cache.ComputeIfPresent(ctx, scope, func(data *llm_dto.BudgetData) (*llm_dto.BudgetData, cache_dto.ComputeAction) {
		data.TotalSpent.SubtractInPlace(cost)
		data.HourlySpent.SubtractInPlace(cost)
		data.DailySpent.SubtractInPlace(cost)
		data.LastUpdated = s.clock.Now()

		return data, cache_dto.ComputeActionSet
	})
	if err != nil {
		return fmt.Errorf("unreserving budget cost for scope %q: %w", scope, err)
	}

	return nil
}

// GetStatus returns current budget status for a scope.
//
// Takes scope (string) which identifies the budget scope.
//
// Returns *llm_dto.BudgetStatus containing the current state.
// Returns error if the context is cancelled.
func (s *Store) GetStatus(ctx context.Context, scope string) (*llm_dto.BudgetStatus, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	data, found, err := s.cache.GetIfPresent(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("retrieving budget status for scope %q: %w", scope, err)
	}
	if !found || data == nil {
		return &llm_dto.BudgetStatus{
			Scope:           scope,
			TotalSpent:      maths.ZeroMoney(llm_dto.CostCurrency),
			DailySpent:      maths.ZeroMoney(llm_dto.CostCurrency),
			HourlySpent:     maths.ZeroMoney(llm_dto.CostCurrency),
			RemainingBudget: maths.ZeroMoney(llm_dto.CostCurrency),
			LastUpdated:     s.clock.Now(),
		}, nil
	}

	now := s.clock.Now()
	hourlySpent := data.HourlySpent
	dailySpent := data.DailySpent

	if now.Sub(data.HourStart) >= time.Hour {
		hourlySpent = maths.ZeroMoney(llm_dto.CostCurrency)
	}

	if !isSameDay(now, data.DayStart) {
		dailySpent = maths.ZeroMoney(llm_dto.CostCurrency)
	}

	return &llm_dto.BudgetStatus{
		Scope:           scope,
		TotalSpent:      data.TotalSpent,
		DailySpent:      dailySpent,
		HourlySpent:     hourlySpent,
		RemainingBudget: maths.ZeroMoney(llm_dto.CostCurrency),
		RequestCount:    data.RequestCount,
		TokenCount:      data.TokenCount,
		LastUpdated:     data.LastUpdated,
	}, nil
}

// IncrementRequests atomically increments the request count.
//
// Takes scope (string) which identifies the budget scope.
// Takes count (int64) which is the number to add.
//
// Returns error if the context is cancelled.
func (s *Store) IncrementRequests(ctx context.Context, scope string, count int64) error {
	return s.incrementCounter(ctx, scope, func(data *llm_dto.BudgetData) { data.RequestCount += count })
}

// IncrementTokens atomically increments the token count for a scope.
//
// Takes scope (string) which identifies the budget scope.
// Takes count (int64) which is the number to add.
//
// Returns error if the context is cancelled.
func (s *Store) IncrementTokens(ctx context.Context, scope string, count int64) error {
	return s.incrementCounter(ctx, scope, func(data *llm_dto.BudgetData) { data.TokenCount += count })
}

// Reset resets counters for a scope.
//
// Takes scope (string) which identifies the budget scope.
//
// Returns error if the context is cancelled.
func (s *Store) Reset(ctx context.Context, scope string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return s.cache.Invalidate(ctx, scope)
}

// incrementCounter is a shared helper for IncrementRequests and
// IncrementTokens.
//
// Takes scope (string) which identifies the budget scope.
// Takes mutate (func(*llm_dto.BudgetData)) which applies the
// counter mutation.
//
// Returns error if the context is cancelled or the compute fails.
func (s *Store) incrementCounter(ctx context.Context, scope string, mutate func(*llm_dto.BudgetData)) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	now := s.clock.Now()
	_, _, err := s.cache.Compute(ctx, scope, func(old *llm_dto.BudgetData, found bool) (*llm_dto.BudgetData, cache_dto.ComputeAction) {
		data := getOrCreate(old, found, now)
		mutate(data)
		data.LastUpdated = now
		return data, cache_dto.ComputeActionSet
	})
	if err != nil {
		return fmt.Errorf("incrementing budget counter for scope %q: %w", scope, err)
	}

	return nil
}

// New creates a new cache-backed budget store.
//
// Takes config (Config) which configures the store.
//
// Returns *Store which is ready for use.
// Returns error when cache creation fails.
func New(ctx context.Context, config Config) (*Store, error) {
	if config.CacheService == nil {
		return nil, errors.New("cache service is required")
	}

	namespace := config.Namespace
	if namespace == "" {
		namespace = DefaultNamespace
	}

	maximumSize := config.MaximumSize
	if maximumSize <= 0 {
		maximumSize = DefaultMaximumSize
	}

	c, err := cache_domain.NewCacheBuilder[string, *llm_dto.BudgetData](config.CacheService).
		FactoryBlueprint(FactoryBlueprintName).
		Namespace(namespace).
		MaximumSize(maximumSize).
		Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create budget cache: %w", err)
	}

	clk := config.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	return &Store{
		cache: c,
		clock: clk,
	}, nil
}

// getOrCreate returns the existing data or creates a new zero-valued
// entry.
//
// Takes old (*llm_dto.BudgetData) which is the existing data if any.
// Takes found (bool) which indicates whether old is valid.
// Takes now (time.Time) which is the current time for window starts.
//
// Returns *llm_dto.BudgetData which is the existing or newly created
// entry.
func getOrCreate(old *llm_dto.BudgetData, found bool, now time.Time) *llm_dto.BudgetData {
	if found && old != nil {
		return old
	}
	return &llm_dto.BudgetData{
		TotalSpent:  maths.ZeroMoney(llm_dto.CostCurrency),
		HourlySpent: maths.ZeroMoney(llm_dto.CostCurrency),
		DailySpent:  maths.ZeroMoney(llm_dto.CostCurrency),
		HourStart:   now.Truncate(time.Hour),
		DayStart:    truncateToDay(now),
		LastUpdated: now,
	}
}

// resetStaleWindows zeroes hourly and daily counters when their time
// windows have elapsed.
//
// Takes data (*llm_dto.BudgetData) which is the budget data to
// update in place.
// Takes now (time.Time) which is the current time for window checks.
func resetStaleWindows(data *llm_dto.BudgetData, now time.Time) {
	if now.Sub(data.HourStart) >= time.Hour {
		data.HourlySpent = maths.ZeroMoney(llm_dto.CostCurrency)
		data.HourStart = now.Truncate(time.Hour)
	}

	if !isSameDay(now, data.DayStart) {
		data.DailySpent = maths.ZeroMoney(llm_dto.CostCurrency)
		data.DayStart = truncateToDay(now)
	}
}

// checkLimits verifies the estimated cost against all configured
// budget limits.
//
// Takes data (*llm_dto.BudgetData) which holds current spend
// counters.
// Takes estimatedCost (maths.Money) which is the cost to check.
// Takes limits (llm_dto.BudgetLimits) which carries the spend caps.
//
// Returns error which is llm_domain.ErrBudgetExceeded if any limit
// would be breached.
func checkLimits(data *llm_dto.BudgetData, estimatedCost maths.Money, limits llm_dto.BudgetLimits) error {
	if !limits.MaxTotalSpend.CheckIsZero() {
		projected := data.TotalSpent.Add(estimatedCost)
		if projected.CheckGreaterThan(limits.MaxTotalSpend) {
			return llm_domain.ErrBudgetExceeded
		}
	}

	if !limits.MaxDailySpend.CheckIsZero() {
		projected := data.DailySpent.Add(estimatedCost)
		if projected.CheckGreaterThan(limits.MaxDailySpend) {
			return llm_domain.ErrBudgetExceeded
		}
	}

	if !limits.MaxHourlySpend.CheckIsZero() {
		projected := data.HourlySpent.Add(estimatedCost)
		if projected.CheckGreaterThan(limits.MaxHourlySpend) {
			return llm_domain.ErrBudgetExceeded
		}
	}

	return nil
}

// isSameDay reports whether t1 and t2 are on the same calendar day
// in UTC.
//
// Takes t1 (time.Time) which is the first time to compare.
// Takes t2 (time.Time) which is the second time to compare.
//
// Returns bool which is true if both times fall on the same UTC day.
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.UTC().Date()
	y2, m2, d2 := t2.UTC().Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// truncateToDay truncates a time to the start of the day in UTC.
//
// Takes t (time.Time) which is the time to truncate.
//
// Returns time.Time which is midnight UTC on the same day.
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
