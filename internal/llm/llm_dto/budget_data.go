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

// BudgetData is the serialisable representation of budget state for a single
// scope. It is stored as the cache value in the cache-backed budget store.
type BudgetData struct {
	// HourStart is when the current hour window began; resets each hour.
	HourStart time.Time `json:"hourStart"`

	// DayStart is the start of the current day for tracking daily spending.
	DayStart time.Time `json:"dayStart"`

	// LastUpdated is when any budget data was last modified.
	LastUpdated time.Time `json:"lastUpdated"`

	// TotalSpent is the cumulative amount spent for this scope since tracking began.
	TotalSpent maths.Money `json:"totalSpent"`

	// HourlySpent tracks the total amount spent during the current hour.
	HourlySpent maths.Money `json:"hourlySpent"`

	// DailySpent tracks spending since the start of the current day.
	DailySpent maths.Money `json:"dailySpent"`

	// RequestCount is the total number of requests made within this budget scope.
	RequestCount int64 `json:"requestCount"`

	// TokenCount is the total number of tokens used within the current budget period.
	TokenCount int64 `json:"tokenCount"`
}
