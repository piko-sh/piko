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

package llm_dto

import (
	"time"

	"piko.sh/piko/wdk/maths"
)

// LimitType defines the kind of limit being applied.
type LimitType string

const (
	// LimitTypeSpend is the limit type for spending amounts in USD.
	LimitTypeSpend LimitType = "spend"

	// LimitTypeRequests limits by the number of requests made.
	LimitTypeRequests LimitType = "requests"

	// LimitTypeTokens is a limit type that restricts the number of tokens.
	LimitTypeTokens LimitType = "tokens"
)

// BudgetConfig defines spending limits for a scope.
// A zero value in any limit field means no limit is set.
type BudgetConfig struct {
	// OnAlert is called when the alert threshold is reached.
	// This callback runs in a separate goroutine.
	OnAlert func(status BudgetStatus)

	// MaxTotalSpend is the maximum total spend. Zero means unlimited.
	MaxTotalSpend maths.Money

	// MaxDailySpend is the most that can be spent in one day. Zero means no limit.
	MaxDailySpend maths.Money

	// MaxHourlySpend is the maximum hourly spend. Zero means unlimited.
	MaxHourlySpend maths.Money

	// MaxCostPerRequest is the maximum cost for a single
	// request. Zero means unlimited.
	MaxCostPerRequest maths.Money

	// AlertThreshold is the budget fraction at which to trigger an alert.
	// For example, 0.8 triggers at 80% of budget; zero disables alerts.
	AlertThreshold float64
}

// HasTotalSpendLimit reports whether a total spend limit is configured.
//
// Returns bool which is true when a total spend limit has been set.
func (c *BudgetConfig) HasTotalSpendLimit() bool {
	return !c.MaxTotalSpend.CheckIsZero()
}

// HasDailySpendLimit reports whether a daily spend limit is configured.
//
// Returns bool which is true if a daily spend limit is set.
func (c *BudgetConfig) HasDailySpendLimit() bool {
	return !c.MaxDailySpend.CheckIsZero()
}

// HasHourlySpendLimit reports whether an hourly spend limit is configured.
//
// Returns bool which is true if a non-zero hourly spend limit is set.
func (c *BudgetConfig) HasHourlySpendLimit() bool {
	return !c.MaxHourlySpend.CheckIsZero()
}

// HasPerRequestLimit reports whether a per-request cost limit is configured.
//
// Returns bool which is true when MaxCostPerRequest is non-zero.
func (c *BudgetConfig) HasPerRequestLimit() bool {
	return !c.MaxCostPerRequest.CheckIsZero()
}

// HasAlertThreshold reports whether an alert threshold is configured.
//
// Returns bool which is true when the threshold is positive and a handler is
// set.
func (c *BudgetConfig) HasAlertThreshold() bool {
	return c.AlertThreshold > 0 && c.OnAlert != nil
}

// BudgetStatus holds the current spending and usage data for a budget scope.
type BudgetStatus struct {
	// LastUpdated is when this status was last changed.
	LastUpdated time.Time

	// TotalSpent is the cumulative amount spent within this budget scope.
	TotalSpent maths.Money

	// DailySpent is the total amount spent during the current day.
	DailySpent maths.Money

	// HourlySpent is the total amount spent in the current hour.
	HourlySpent maths.Money

	// RemainingBudget is the amount of budget left for the day.
	// Zero if unlimited.
	RemainingBudget maths.Money

	// Scope identifies the budget scope, e.g., "global",
	// "provider:openai", "group:user:123".
	Scope string

	// RequestCount is the total number of requests made.
	RequestCount int64

	// TokenCount is the total number of tokens used.
	TokenCount int64

	// ThresholdReached indicates whether the alert threshold has been reached.
	ThresholdReached bool
}

// BudgetLimits carries the spend limits for an atomic check-and-reserve
// operation. Zero values mean no limit is configured for that window.
type BudgetLimits struct {
	// MaxTotalSpend is the cumulative spend limit.
	MaxTotalSpend maths.Money

	// MaxDailySpend is the maximum spend per day.
	MaxDailySpend maths.Money

	// MaxHourlySpend is the maximum spend per hour.
	MaxHourlySpend maths.Money
}

// Limit represents a single limit configuration.
type Limit struct {
	// Type specifies the kind of limit being applied.
	Type LimitType

	// Value is the limit amount.
	Value maths.Decimal

	// Window is the time period for limits that reset after a set time.
	Window time.Duration
}
