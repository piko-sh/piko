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

package llm_domain

import (
	"context"
	"fmt"
	"sync"

	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/maths"
)

// BudgetStorePort defines the driven port for storing budget data.
// Implementations must be safe for concurrent use.
type BudgetStorePort interface {
	// Record adds a cost entry for a scope.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes scope (string) which identifies the budget scope.
	// Takes cost (*llm_dto.CostEstimate) which is the cost to record.
	//
	// Returns error if the recording fails.
	Record(ctx context.Context, scope string, cost *llm_dto.CostEstimate) error

	// GetStatus returns current budget status for a scope.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes scope (string) which identifies the budget scope.
	//
	// Returns *llm_dto.BudgetStatus containing the current state.
	// Returns error if the status cannot be retrieved.
	GetStatus(ctx context.Context, scope string) (*llm_dto.BudgetStatus, error)

	// CheckAndReserve atomically checks whether the estimated cost fits
	// within all configured limits and, if so, reserves it by adding the
	// cost to the current spend counters.
	//
	// Implementations must hold their lock for the entire
	// check-and-reserve to prevent concurrent overspend.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes scope (string) which identifies the budget scope.
	// Takes estimatedCost (maths.Money) which is the cost to reserve.
	// Takes limits (llm_dto.BudgetLimits) which carries the spend limits.
	//
	// Returns error which is ErrBudgetExceeded if any limit is breached.
	CheckAndReserve(ctx context.Context, scope string, estimatedCost maths.Money, limits llm_dto.BudgetLimits) error

	// Unreserve releases a previously reserved cost when the request fails
	// before RecordUsage is called. This prevents phantom charges.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes scope (string) which identifies the budget scope.
	// Takes cost (maths.Money) which is the cost to release.
	//
	// Returns error if the release fails.
	Unreserve(ctx context.Context, scope string, cost maths.Money) error

	// IncrementRequests atomically increments the request count.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes scope (string) which identifies the budget scope.
	// Takes count (int64) which is the number to add.
	//
	// Returns error if the increment fails.
	IncrementRequests(ctx context.Context, scope string, count int64) error

	// IncrementTokens atomically increments the token count.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes scope (string) which identifies the budget scope.
	// Takes count (int64) which is the number to add.
	//
	// Returns error if the increment fails.
	IncrementTokens(ctx context.Context, scope string, count int64) error

	// Reset resets counters for a scope.
	//
	// Takes ctx (context.Context) which controls cancellation.
	// Takes scope (string) which identifies the budget scope.
	//
	// Returns error if the reset fails.
	Reset(ctx context.Context, scope string) error
}

// BudgetManager controls and tracks spending limits for LLM usage.
type BudgetManager struct {
	// store provides persistence for budget data and usage tracking.
	store BudgetStorePort

	// configs maps scope names to their budget configurations.
	configs map[string]*llm_dto.BudgetConfig

	// calculator computes costs for budget calculations.
	calculator *CostCalculator

	// alertTriggered tracks which scopes have triggered budget alerts.
	alertTriggered map[string]bool

	// mu guards concurrent access to configs and alertTriggered.
	mu sync.RWMutex
}

// NewBudgetManager creates a new budget manager instance.
//
// Takes store (BudgetStorePort) which handles saving and loading budget data.
// Takes calculator (*CostCalculator) which works out costs.
//
// Returns *BudgetManager which is ready for configuration.
func NewBudgetManager(store BudgetStorePort, calculator *CostCalculator) *BudgetManager {
	return &BudgetManager{
		store:          store,
		configs:        make(map[string]*llm_dto.BudgetConfig),
		calculator:     calculator,
		alertTriggered: make(map[string]bool),
		mu:             sync.RWMutex{},
	}
}

// SetBudget configures a budget for a scope.
//
// Takes scope (string) which identifies the budget scope.
// Takes config (*llm_dto.BudgetConfig) which contains the budget limits.
//
// Safe for concurrent use. Resets any previous alert state for the scope.
func (m *BudgetManager) SetBudget(scope string, config *llm_dto.BudgetConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[scope] = config
	delete(m.alertTriggered, scope)
}

// RemoveBudget removes a budget configuration for a scope.
//
// Takes scope (string) which identifies the budget scope to remove.
//
// Safe for concurrent use; protected by mutex.
func (m *BudgetManager) RemoveBudget(scope string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.configs, scope)
	delete(m.alertTriggered, scope)
}

// CheckBudget returns an error if a request would exceed limits.
// This is a pre-request check based on estimated cost.
//
// Takes scope (string) which identifies the budget scope.
// Takes estimatedCost (maths.Money) which is the estimated cost.
//
// Returns error which is ErrBudgetExceeded or ErrMaxCostExceeded if
// limits are exceeded.
//
// Safe for concurrent use; protects internal config access with a mutex.
func (m *BudgetManager) CheckBudget(ctx context.Context, scope string, estimatedCost maths.Money) error {
	m.mu.RLock()
	config := m.configs[scope]
	m.mu.RUnlock()

	if config == nil {
		return nil
	}

	if config.HasPerRequestLimit() {
		if estimatedCost.CheckGreaterThan(config.MaxCostPerRequest) {
			return ErrMaxCostExceeded
		}
	}

	limits := llm_dto.BudgetLimits{
		MaxTotalSpend:  config.MaxTotalSpend,
		MaxDailySpend:  config.MaxDailySpend,
		MaxHourlySpend: config.MaxHourlySpend,
	}

	if err := m.store.CheckAndReserve(ctx, scope, estimatedCost, limits); err != nil {
		return fmt.Errorf("checking budget for scope %q: %w", scope, err)
	}

	return nil
}

// UnreserveBudget releases a previously reserved cost when the request fails
// before RecordUsage is called.
//
// Takes scope (string) which identifies the budget scope.
// Takes cost (maths.Money) which is the cost to release.
//
// Returns error if the release fails.
func (m *BudgetManager) UnreserveBudget(ctx context.Context, scope string, cost maths.Money) error {
	return m.store.Unreserve(ctx, scope, cost)
}

// RecordUsage records actual usage after a request completes.
// Also triggers alert callbacks if thresholds are reached.
//
// Takes scope (string) which identifies the budget scope.
// Takes cost (*llm_dto.CostEstimate) which is the actual cost to record.
//
// Returns error when recording fails.
func (m *BudgetManager) RecordUsage(ctx context.Context, scope string, cost *llm_dto.CostEstimate) error {
	if cost == nil {
		return nil
	}

	if err := m.store.Record(ctx, scope, cost); err != nil {
		return fmt.Errorf("recording usage for scope %q: %w", scope, err)
	}

	if err := m.store.IncrementRequests(ctx, scope, 1); err != nil {
		return fmt.Errorf("incrementing request count for scope %q: %w", scope, err)
	}
	if err := m.store.IncrementTokens(ctx, scope, int64(cost.TotalTokens)); err != nil {
		return fmt.Errorf("incrementing token count for scope %q: %w", scope, err)
	}

	m.checkAlertThreshold(ctx, scope)

	return nil
}

// GetStatus returns the current budget status for a scope.
//
// Takes scope (string) which identifies the budget scope.
//
// Returns *llm_dto.BudgetStatus which contains the current state.
// Returns error when the status cannot be retrieved.
//
// Safe for concurrent use. Uses a read lock when accessing configuration.
func (m *BudgetManager) GetStatus(ctx context.Context, scope string) (*llm_dto.BudgetStatus, error) {
	status, err := m.store.GetStatus(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("getting budget status for scope %q: %w", scope, err)
	}

	m.mu.RLock()
	config := m.configs[scope]
	m.mu.RUnlock()

	if config != nil {
		status.RemainingBudget = minRemainingBudget(config, status)
	}

	return status, nil
}

// HasBudget reports whether a scope has a budget configured.
//
// Takes scope (string) which identifies the budget scope.
//
// Returns bool which is true if a budget is configured for the scope.
//
// Safe for concurrent use.
func (m *BudgetManager) HasBudget(scope string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.configs[scope]
	return exists
}

// GetConfig returns the budget configuration for a scope, or nil if not found.
//
// Takes scope (string) which identifies the budget scope.
//
// Returns *llm_dto.BudgetConfig containing the configuration, or nil if not
// found.
//
// Safe for concurrent use.
func (m *BudgetManager) GetConfig(scope string) *llm_dto.BudgetConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.configs[scope]
}

// Reset resets budget counters for a scope.
//
// Takes scope (string) which identifies the budget scope.
//
// Returns error if the reset fails.
//
// Safe for concurrent use.
func (m *BudgetManager) Reset(ctx context.Context, scope string) error {
	m.mu.Lock()
	delete(m.alertTriggered, scope)
	m.mu.Unlock()

	return m.store.Reset(ctx, scope)
}

// ResetDaily resets the daily spend counter for all scopes.
// This should be called at the start of each day.
//
// Returns error if any reset fails.
//
// Safe for concurrent use. Uses mutex to protect config and alert state access.
func (m *BudgetManager) ResetDaily(ctx context.Context) error {
	m.mu.RLock()
	scopes := make([]string, 0, len(m.configs))
	for scope := range m.configs {
		scopes = append(scopes, scope)
	}
	m.mu.RUnlock()

	for _, scope := range scopes {
		if err := m.store.Reset(ctx, scope); err != nil {
			return fmt.Errorf("resetting daily budget for scope %q: %w", scope, err)
		}
		m.mu.Lock()
		delete(m.alertTriggered, scope)
		m.mu.Unlock()
	}

	return nil
}

// checkAlertThreshold checks if alert threshold is reached and triggers
// callback.
//
// Takes scope (string) which identifies the budget scope to check.
//
// Safe for concurrent use. When threshold is reached, the alert
// callback runs in a separate goroutine.
func (m *BudgetManager) checkAlertThreshold(ctx context.Context, scope string) {
	ctx, l := logger_domain.From(ctx, log)
	m.mu.RLock()
	config := m.configs[scope]
	alreadyTriggered := m.alertTriggered[scope]
	m.mu.RUnlock()

	if config == nil || !config.HasAlertThreshold() || alreadyTriggered {
		return
	}

	status, err := m.GetStatus(ctx, scope)
	if err != nil {
		return
	}

	if !isSpendThresholdReached(config, status) {
		return
	}

	m.mu.Lock()
	m.alertTriggered[scope] = true
	m.mu.Unlock()

	status.ThresholdReached = true
	m.fireAlertCallback(l, scope, config, status)
}

// fireAlertCallback invokes the budget alert callback in a separate goroutine
// with panic recovery.
//
// Takes l (logger_domain.Logger) which provides structured logging.
// Takes scope (string) which identifies the budget scope.
// Takes config (*llm_dto.BudgetConfig) which provides the alert callback.
// Takes status (*llm_dto.BudgetStatus) which is passed to the callback.
func (*BudgetManager) fireAlertCallback(l logger_domain.Logger, scope string, config *llm_dto.BudgetConfig, status *llm_dto.BudgetStatus) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.Error("Budget alert callback panicked",
					logger_domain.String("scope", scope),
					logger_domain.String("panic", fmt.Sprint(r)),
				)
			}
		}()
		config.OnAlert(*status)
	}()
}

// EstimateInputTokens provides a rough estimate of input tokens based on
// message content. It uses the common approximation of ~4 characters per
// token.
//
// Takes messages ([]llm_dto.Message) which are the messages to estimate.
//
// Returns int which is the estimated number of input tokens.
func EstimateInputTokens(messages []llm_dto.Message) int {
	var totalChars int
	for _, message := range messages {
		totalChars += len(message.Content)
	}
	return totalChars / CharactersPerToken
}

// minRemainingBudget computes the minimum remaining budget across all
// configured limits (total, daily, hourly).
//
// Takes config (*llm_dto.BudgetConfig) which holds the spend limits.
// Takes status (*llm_dto.BudgetStatus) which holds current spend
// amounts.
//
// Returns maths.Money which is the smallest remaining budget across
// all active limits.
func minRemainingBudget(config *llm_dto.BudgetConfig, status *llm_dto.BudgetStatus) maths.Money {
	var result maths.Money
	first := true

	if config.HasTotalSpendLimit() {
		result = config.MaxTotalSpend.Subtract(status.TotalSpent)
		first = false
	}
	if config.HasDailySpendLimit() {
		remaining := config.MaxDailySpend.Subtract(status.DailySpent)
		if first || remaining.CheckLessThan(result) {
			result = remaining
			first = false
		}
	}
	if config.HasHourlySpendLimit() {
		remaining := config.MaxHourlySpend.Subtract(status.HourlySpent)
		if first || remaining.CheckLessThan(result) {
			result = remaining
		}
	}

	return result
}

// isSpendThresholdReached checks whether any spend limit has reached its alert
// threshold.
//
// Takes config (*llm_dto.BudgetConfig) which holds the spend limits and
// threshold.
// Takes status (*llm_dto.BudgetStatus) which holds the current spend amounts.
//
// Returns bool which is true when at least one limit has reached the threshold.
func isSpendThresholdReached(config *llm_dto.BudgetConfig, status *llm_dto.BudgetStatus) bool {
	if config.HasDailySpendLimit() {
		thresholdAmount := config.MaxDailySpend.MultiplyFloat(config.AlertThreshold)
		if !status.DailySpent.CheckLessThan(thresholdAmount) {
			return true
		}
	}
	if config.HasTotalSpendLimit() {
		thresholdAmount := config.MaxTotalSpend.MultiplyFloat(config.AlertThreshold)
		if !status.TotalSpent.CheckLessThan(thresholdAmount) {
			return true
		}
	}
	return false
}
