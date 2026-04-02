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
	"time"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

// budgetData stores budget state for a single scope.
type budgetData struct {
	// hourStart is when the current hour window began; resets each hour.
	hourStart time.Time

	// dayStart is the start of the current day for tracking daily spending.
	dayStart time.Time

	// lastUpdated is when any budget data was last modified.
	lastUpdated time.Time

	// totalSpent is the cumulative amount spent for this scope since tracking began.
	totalSpent maths.Money

	// hourlySpent tracks the total amount spent during the current hour.
	hourlySpent maths.Money

	// dailySpent tracks spending since the start of the current day.
	dailySpent maths.Money

	// requestCount is the total number of requests made within this budget scope.
	requestCount int64

	// tokenCount is the total number of tokens used within the current budget period.
	tokenCount int64
}

// Store is an in-memory implementation of BudgetStorePort.
type Store struct {
	// clock provides the current time for expiry checks and timestamps.
	clock clock.Clock

	// data maps scope names to their budget tracking information.
	data map[string]*budgetData

	// mu guards access to the store's mutable fields.
	mu sync.RWMutex
}

var _ llm_domain.BudgetStorePort = (*Store)(nil)

// StoreOption is a function that sets up the Store.
type StoreOption func(*Store)

// Record adds a cost entry for a scope.
//
// Takes scope (string) which identifies the budget scope.
// Takes cost (*llm_dto.CostEstimate) which is the cost to record.
//
// Returns error if the recording fails (always nil for in-memory).
//
// Safe for concurrent use; protects internal state with a mutex.
func (s *Store) Record(ctx context.Context, scope string, cost *llm_dto.CostEstimate) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if cost == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data := s.getOrCreateData(scope)
	now := s.clock.Now()

	if now.Sub(data.hourStart) >= time.Hour {
		data.hourlySpent = maths.ZeroMoney(llm_dto.CostCurrency)
		data.hourStart = now.Truncate(time.Hour)
	}

	if !isSameDay(now, data.dayStart) {
		data.dailySpent = maths.ZeroMoney(llm_dto.CostCurrency)
		data.dayStart = truncateToDay(now)
	}

	data.totalSpent.AddInPlace(cost.TotalCost)
	data.hourlySpent.AddInPlace(cost.TotalCost)
	data.dailySpent.AddInPlace(cost.TotalCost)
	data.lastUpdated = now

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
//
// Safe for concurrent use; holds the write lock for the entire operation.
func (s *Store) CheckAndReserve(ctx context.Context, scope string, estimatedCost maths.Money, limits llm_dto.BudgetLimits) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data := s.getOrCreateData(scope)
	now := s.clock.Now()

	if now.Sub(data.hourStart) >= time.Hour {
		data.hourlySpent = maths.ZeroMoney(llm_dto.CostCurrency)
		data.hourStart = now.Truncate(time.Hour)
	}

	if !isSameDay(now, data.dayStart) {
		data.dailySpent = maths.ZeroMoney(llm_dto.CostCurrency)
		data.dayStart = truncateToDay(now)
	}

	if !limits.MaxTotalSpend.CheckIsZero() {
		projected := data.totalSpent.Add(estimatedCost)
		if projected.CheckGreaterThan(limits.MaxTotalSpend) {
			return llm_domain.ErrBudgetExceeded
		}
	}

	if !limits.MaxDailySpend.CheckIsZero() {
		projected := data.dailySpent.Add(estimatedCost)
		if projected.CheckGreaterThan(limits.MaxDailySpend) {
			return llm_domain.ErrBudgetExceeded
		}
	}

	if !limits.MaxHourlySpend.CheckIsZero() {
		projected := data.hourlySpent.Add(estimatedCost)
		if projected.CheckGreaterThan(limits.MaxHourlySpend) {
			return llm_domain.ErrBudgetExceeded
		}
	}

	data.totalSpent.AddInPlace(estimatedCost)
	data.hourlySpent.AddInPlace(estimatedCost)
	data.dailySpent.AddInPlace(estimatedCost)
	data.lastUpdated = now

	return nil
}

// Unreserve releases a previously reserved cost when the request fails before
// the actual usage is recorded.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes scope (string) which identifies the budget scope.
// Takes cost (maths.Money) which is the reserved cost to release.
//
// Returns error if the context is cancelled.
//
// Safe for concurrent use; protected by mutex.
func (s *Store) Unreserve(ctx context.Context, scope string, cost maths.Money) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data := s.data[scope]
	if data == nil {
		return nil
	}

	data.totalSpent.SubtractInPlace(cost)
	data.hourlySpent.SubtractInPlace(cost)
	data.dailySpent.SubtractInPlace(cost)
	data.lastUpdated = s.clock.Now()

	return nil
}

// GetStatus returns current budget status for a scope.
//
// Takes scope (string) which identifies the budget scope.
//
// Returns *llm_dto.BudgetStatus containing the current state.
// Returns error if the status cannot be retrieved.
//
// Safe for concurrent use. Uses a read lock to access budget data.
func (s *Store) GetStatus(ctx context.Context, scope string) (*llm_dto.BudgetStatus, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data := s.data[scope]

	if data == nil {
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
	hourlySpent := data.hourlySpent
	dailySpent := data.dailySpent

	if now.Sub(data.hourStart) >= time.Hour {
		hourlySpent = maths.ZeroMoney(llm_dto.CostCurrency)
	}

	if !isSameDay(now, data.dayStart) {
		dailySpent = maths.ZeroMoney(llm_dto.CostCurrency)
	}

	return &llm_dto.BudgetStatus{
		Scope:           scope,
		TotalSpent:      data.totalSpent,
		DailySpent:      dailySpent,
		HourlySpent:     hourlySpent,
		RemainingBudget: maths.ZeroMoney(llm_dto.CostCurrency),
		RequestCount:    data.requestCount,
		TokenCount:      data.tokenCount,
		LastUpdated:     data.lastUpdated,
	}, nil
}

// IncrementRequests atomically increments the request count.
//
// Takes scope (string) which identifies the budget scope.
// Takes count (int64) which is the number to add.
//
// Returns error if the increment fails (always nil for in-memory).
//
// Safe for concurrent use; protected by mutex.
func (s *Store) IncrementRequests(ctx context.Context, scope string, count int64) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data := s.getOrCreateData(scope)
	data.requestCount += count
	data.lastUpdated = s.clock.Now()

	return nil
}

// IncrementTokens atomically increments the token count for a scope.
//
// Takes scope (string) which identifies the budget scope.
// Takes count (int64) which is the number to add.
//
// Returns error when the context is cancelled.
//
// Safe for concurrent use. Protected by a mutex.
func (s *Store) IncrementTokens(ctx context.Context, scope string, count int64) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data := s.getOrCreateData(scope)
	data.tokenCount += count
	data.lastUpdated = s.clock.Now()

	return nil
}

// Reset resets counters for a scope.
//
// Takes scope (string) which identifies the budget scope.
//
// Returns error if the reset fails (always nil for in-memory).
//
// Safe for concurrent use; protected by a mutex.
func (s *Store) Reset(ctx context.Context, scope string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, scope)
	return nil
}

// ResetAll resets all budget data across all scopes.
// Intended for testing.
//
// Returns error if the reset fails (always nil for in-memory).
//
// Safe for concurrent use.
func (s *Store) ResetAll(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]*budgetData)
	return nil
}

// getOrCreateData returns budget data for a scope, creating it if needed.
// Must be called with lock held.
//
// Takes scope (string) which identifies the budget scope to retrieve or create.
//
// Returns *budgetData which contains the budget tracking data for the scope.
func (s *Store) getOrCreateData(scope string) *budgetData {
	data := s.data[scope]
	if data == nil {
		now := s.clock.Now()
		data = &budgetData{
			totalSpent:  maths.ZeroMoney(llm_dto.CostCurrency),
			hourlySpent: maths.ZeroMoney(llm_dto.CostCurrency),
			dailySpent:  maths.ZeroMoney(llm_dto.CostCurrency),
			hourStart:   now.Truncate(time.Hour),
			dayStart:    truncateToDay(now),
			lastUpdated: now,
		}
		s.data[scope] = data
	}
	return data
}

// WithClock sets the clock used for time operations.
// If not set, clock.RealClock() is used.
//
// Takes c (clock.Clock) which provides time operations.
//
// Returns StoreOption which applies this setting to the store.
func WithClock(c clock.Clock) StoreOption {
	return func(s *Store) {
		s.clock = c
	}
}

// New creates a new in-memory budget store.
//
// Takes opts (...StoreOption) which are optional functions to configure the
// store.
//
// Returns *Store which is ready for use.
func New(opts ...StoreOption) *Store {
	s := &Store{
		clock: clock.RealClock(),
		data:  make(map[string]*budgetData),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// isSameDay reports whether t1 and t2 are on the same calendar day in UTC.
//
// Takes t1 (time.Time) which is the first time to compare.
// Takes t2 (time.Time) which is the second time to compare.
//
// Returns bool which is true if both times fall on the same UTC date.
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.UTC().Date()
	y2, m2, d2 := t2.UTC().Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// truncateToDay truncates a time to the start of the day in UTC.
//
// Takes t (time.Time) which is the time to truncate.
//
// Returns time.Time which is the input time with hours, minutes, seconds, and
// nanoseconds set to zero.
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
