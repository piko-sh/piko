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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/maths"
)

func TestBudgetData_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 1, 14, 30, 0, 0, time.UTC)

	original := &BudgetData{
		HourStart:    now.Truncate(time.Hour),
		DayStart:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		LastUpdated:  now,
		TotalSpent:   maths.NewMoneyFromString("42.50", CostCurrency),
		HourlySpent:  maths.NewMoneyFromString("5.25", CostCurrency),
		DailySpent:   maths.NewMoneyFromString("18.75", CostCurrency),
		RequestCount: 150,
		TokenCount:   98000,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded BudgetData
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.True(t, original.TotalSpent.MustEquals(decoded.TotalSpent))
	assert.True(t, original.HourlySpent.MustEquals(decoded.HourlySpent))
	assert.True(t, original.DailySpent.MustEquals(decoded.DailySpent))
	assert.Equal(t, original.RequestCount, decoded.RequestCount)
	assert.Equal(t, original.TokenCount, decoded.TokenCount)
	assert.True(t, original.HourStart.Equal(decoded.HourStart))
	assert.True(t, original.DayStart.Equal(decoded.DayStart))
	assert.True(t, original.LastUpdated.Equal(decoded.LastUpdated))
}
