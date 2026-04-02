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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/maths"
)

const testCurrency = "USD"

func TestNewBudgetManager(t *testing.T) {
	store := NewMockBudgetStore()
	calc := NewCostCalculator()

	manager := NewBudgetManager(store, calc)

	require.NotNil(t, manager)
	assert.False(t, manager.HasBudget("any-scope"))
}

func TestBudgetManager_SetAndRemoveBudget(t *testing.T) {
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)

	scope := "test-scope"
	config := &llm_dto.BudgetConfig{
		MaxDailySpend: maths.NewMoneyFromString("100.00", testCurrency),
	}

	t.Run("set budget", func(t *testing.T) {
		assert.False(t, manager.HasBudget(scope))

		manager.SetBudget(scope, config)

		assert.True(t, manager.HasBudget(scope))
		assert.Equal(t, config, manager.GetConfig(scope))
	})

	t.Run("remove budget", func(t *testing.T) {
		manager.RemoveBudget(scope)

		assert.False(t, manager.HasBudget(scope))
		assert.Nil(t, manager.GetConfig(scope))
	})
}

func TestBudgetManager_CheckBudget(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		wantErr       error
		config        *llm_dto.BudgetConfig
		currentSpend  maths.Money
		estimatedCost maths.Money
		name          string
	}{
		{
			name:          "no budget configured allows all",
			config:        nil,
			estimatedCost: maths.NewMoneyFromString("100.00", testCurrency),
			wantErr:       nil,
		},
		{
			name: "within per-request limit",
			config: &llm_dto.BudgetConfig{
				MaxCostPerRequest: maths.NewMoneyFromString("10.00", testCurrency),
			},
			estimatedCost: maths.NewMoneyFromString("5.00", testCurrency),
			wantErr:       nil,
		},
		{
			name: "exceeds per-request limit",
			config: &llm_dto.BudgetConfig{
				MaxCostPerRequest: maths.NewMoneyFromString("10.00", testCurrency),
			},
			estimatedCost: maths.NewMoneyFromString("15.00", testCurrency),
			wantErr:       ErrMaxCostExceeded,
		},
		{
			name: "within daily limit",
			config: &llm_dto.BudgetConfig{
				MaxDailySpend: maths.NewMoneyFromString("100.00", testCurrency),
			},
			currentSpend:  maths.NewMoneyFromString("50.00", testCurrency),
			estimatedCost: maths.NewMoneyFromString("30.00", testCurrency),
			wantErr:       nil,
		},
		{
			name: "exceeds daily limit",
			config: &llm_dto.BudgetConfig{
				MaxDailySpend: maths.NewMoneyFromString("100.00", testCurrency),
			},
			currentSpend:  maths.NewMoneyFromString("80.00", testCurrency),
			estimatedCost: maths.NewMoneyFromString("30.00", testCurrency),
			wantErr:       ErrBudgetExceeded,
		},
		{
			name: "within hourly limit",
			config: &llm_dto.BudgetConfig{
				MaxHourlySpend: maths.NewMoneyFromString("20.00", testCurrency),
			},
			currentSpend:  maths.NewMoneyFromString("10.00", testCurrency),
			estimatedCost: maths.NewMoneyFromString("5.00", testCurrency),
			wantErr:       nil,
		},
		{
			name: "exceeds hourly limit",
			config: &llm_dto.BudgetConfig{
				MaxHourlySpend: maths.NewMoneyFromString("20.00", testCurrency),
			},
			currentSpend:  maths.NewMoneyFromString("15.00", testCurrency),
			estimatedCost: maths.NewMoneyFromString("10.00", testCurrency),
			wantErr:       ErrBudgetExceeded,
		},
		{
			name: "within total limit",
			config: &llm_dto.BudgetConfig{
				MaxTotalSpend: maths.NewMoneyFromString("1000.00", testCurrency),
			},
			currentSpend:  maths.NewMoneyFromString("500.00", testCurrency),
			estimatedCost: maths.NewMoneyFromString("100.00", testCurrency),
			wantErr:       nil,
		},
		{
			name: "exceeds total limit",
			config: &llm_dto.BudgetConfig{
				MaxTotalSpend: maths.NewMoneyFromString("1000.00", testCurrency),
			},
			currentSpend:  maths.NewMoneyFromString("950.00", testCurrency),
			estimatedCost: maths.NewMoneyFromString("100.00", testCurrency),
			wantErr:       ErrBudgetExceeded,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewMockBudgetStore()
			calc := NewCostCalculator()
			manager := NewBudgetManager(store, calc)
			scope := "test-scope"

			if tc.config != nil {
				manager.SetBudget(scope, tc.config)
			}

			if tc.currentSpend.MustIsPositive() {
				store.mu.Lock()
				store.statuses[scope] = &llm_dto.BudgetStatus{
					Scope:       scope,
					TotalSpent:  tc.currentSpend,
					DailySpent:  tc.currentSpend,
					HourlySpent: tc.currentSpend,
				}
				store.mu.Unlock()
			}

			err := manager.CheckBudget(ctx, scope, tc.estimatedCost)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBudgetManager_RecordUsage(t *testing.T) {
	ctx := context.Background()

	t.Run("nil cost does nothing", func(t *testing.T) {
		store := NewMockBudgetStore()
		calc := NewCostCalculator()
		manager := NewBudgetManager(store, calc)

		err := manager.RecordUsage(ctx, "test-scope", nil)

		assert.NoError(t, err)
	})

	t.Run("records cost and increments counters", func(t *testing.T) {
		store := NewMockBudgetStore()
		calc := NewCostCalculator()
		manager := NewBudgetManager(store, calc)
		scope := "test-scope"

		cost := &llm_dto.CostEstimate{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
			TotalCost:    maths.NewMoneyFromString("0.50", testCurrency),
		}

		err := manager.RecordUsage(ctx, scope, cost)

		require.NoError(t, err)

		status, err := store.GetStatus(ctx, scope)
		require.NoError(t, err)
		assert.Equal(t, int64(1), status.RequestCount)
		assert.Equal(t, int64(150), status.TokenCount)
	})
}

func TestBudgetManager_GetStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("returns status from store", func(t *testing.T) {
		store := NewMockBudgetStore()
		calc := NewCostCalculator()
		manager := NewBudgetManager(store, calc)
		scope := "test-scope"

		store.mu.Lock()
		store.statuses[scope] = &llm_dto.BudgetStatus{
			Scope:      scope,
			TotalSpent: maths.NewMoneyFromString("50.00", testCurrency),
			DailySpent: maths.NewMoneyFromString("10.00", testCurrency),
		}
		store.mu.Unlock()

		status, err := manager.GetStatus(ctx, scope)

		require.NoError(t, err)
		assert.True(t, status.TotalSpent.MustEquals(maths.NewMoneyFromString("50.00", testCurrency)))
		assert.True(t, status.DailySpent.MustEquals(maths.NewMoneyFromString("10.00", testCurrency)))
	})

	t.Run("calculates remaining budget when daily limit configured", func(t *testing.T) {
		store := NewMockBudgetStore()
		calc := NewCostCalculator()
		manager := NewBudgetManager(store, calc)
		scope := "test-scope"

		manager.SetBudget(scope, &llm_dto.BudgetConfig{
			MaxDailySpend: maths.NewMoneyFromString("100.00", testCurrency),
		})

		store.mu.Lock()
		store.statuses[scope] = &llm_dto.BudgetStatus{
			Scope:      scope,
			DailySpent: maths.NewMoneyFromString("40.00", testCurrency),
		}
		store.mu.Unlock()

		status, err := manager.GetStatus(ctx, scope)

		require.NoError(t, err)

		assert.True(t, status.RemainingBudget.MustEquals(maths.NewMoneyFromString("60.00", testCurrency)))
	})
}

func TestBudgetManager_Reset(t *testing.T) {
	ctx := context.Background()

	t.Run("resets budget for scope", func(t *testing.T) {
		store := NewMockBudgetStore()
		calc := NewCostCalculator()
		manager := NewBudgetManager(store, calc)
		scope := "test-scope"

		store.mu.Lock()
		store.statuses[scope] = &llm_dto.BudgetStatus{
			Scope:        scope,
			TotalSpent:   maths.NewMoneyFromString("100.00", testCurrency),
			DailySpent:   maths.NewMoneyFromString("50.00", testCurrency),
			RequestCount: 10,
		}
		store.mu.Unlock()

		err := manager.Reset(ctx, scope)

		require.NoError(t, err)

		store.mu.Lock()
		_, exists := store.statuses[scope]
		store.mu.Unlock()
		assert.False(t, exists)
	})
}

func TestBudgetManager_ResetDaily(t *testing.T) {
	ctx := context.Background()

	t.Run("resets all configured scopes", func(t *testing.T) {
		store := NewMockBudgetStore()
		calc := NewCostCalculator()
		manager := NewBudgetManager(store, calc)

		manager.SetBudget("scope1", &llm_dto.BudgetConfig{
			MaxDailySpend: maths.NewMoneyFromString("100.00", testCurrency),
		})
		manager.SetBudget("scope2", &llm_dto.BudgetConfig{
			MaxDailySpend: maths.NewMoneyFromString("200.00", testCurrency),
		})

		store.mu.Lock()
		store.statuses["scope1"] = &llm_dto.BudgetStatus{
			Scope:      "scope1",
			DailySpent: maths.NewMoneyFromString("50.00", testCurrency),
		}
		store.statuses["scope2"] = &llm_dto.BudgetStatus{
			Scope:      "scope2",
			DailySpent: maths.NewMoneyFromString("100.00", testCurrency),
		}
		store.mu.Unlock()

		err := manager.ResetDaily(ctx)

		require.NoError(t, err)

		store.mu.Lock()
		_, exists1 := store.statuses["scope1"]
		_, exists2 := store.statuses["scope2"]
		store.mu.Unlock()
		assert.False(t, exists1)
		assert.False(t, exists2)
	})
}

func TestEstimateInputTokens(t *testing.T) {
	testCases := []struct {
		name     string
		messages []llm_dto.Message
		want     int
	}{
		{
			name:     "empty messages",
			messages: []llm_dto.Message{},
			want:     0,
		},
		{
			name: "single short message",
			messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hi"},
			},

			want: 0,
		},
		{
			name: "single message with 16 chars",
			messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello World!    "},
			},

			want: 4,
		},
		{
			name: "multiple messages",
			messages: []llm_dto.Message{
				{Role: llm_dto.RoleSystem, Content: "You are helpful."},
				{Role: llm_dto.RoleUser, Content: "What is 2+2?"},
			},

			want: 7,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := EstimateInputTokens(tc.messages)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBudgetManager_AlertThreshold(t *testing.T) {
	ctx := context.Background()

	t.Run("triggers alert when threshold reached", func(t *testing.T) {
		store := NewMockBudgetStore()
		calc := NewCostCalculator()
		manager := NewBudgetManager(store, calc)
		scope := "test-scope"

		var alertCalled bool
		var alertStatus llm_dto.BudgetStatus
		var alertMu sync.Mutex

		manager.SetBudget(scope, &llm_dto.BudgetConfig{
			MaxDailySpend:  maths.NewMoneyFromString("100.00", testCurrency),
			AlertThreshold: 0.8,
			OnAlert: func(status llm_dto.BudgetStatus) {
				alertMu.Lock()
				alertCalled = true
				alertStatus = status
				alertMu.Unlock()
			},
		})

		cost := &llm_dto.CostEstimate{
			TotalTokens: 100,
			TotalCost:   maths.NewMoneyFromString("85.00", testCurrency),
		}

		err := manager.RecordUsage(ctx, scope, cost)
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)

		alertMu.Lock()
		assert.True(t, alertCalled, "alert should have been triggered")
		assert.True(t, alertStatus.ThresholdReached)
		alertMu.Unlock()
	})

	t.Run("does not trigger alert below threshold", func(t *testing.T) {
		store := NewMockBudgetStore()
		calc := NewCostCalculator()
		manager := NewBudgetManager(store, calc)
		scope := "test-scope"

		var alertCalled bool
		var alertMu sync.Mutex

		manager.SetBudget(scope, &llm_dto.BudgetConfig{
			MaxDailySpend:  maths.NewMoneyFromString("100.00", testCurrency),
			AlertThreshold: 0.8,
			OnAlert: func(status llm_dto.BudgetStatus) {
				alertMu.Lock()
				alertCalled = true
				alertMu.Unlock()
			},
		})

		cost := &llm_dto.CostEstimate{
			TotalTokens: 100,
			TotalCost:   maths.NewMoneyFromString("50.00", testCurrency),
		}

		err := manager.RecordUsage(ctx, scope, cost)
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)

		alertMu.Lock()
		assert.False(t, alertCalled, "alert should not have been triggered")
		alertMu.Unlock()
	})

	t.Run("only triggers alert once per scope", func(t *testing.T) {
		store := NewMockBudgetStore()
		calc := NewCostCalculator()
		manager := NewBudgetManager(store, calc)
		scope := "test-scope"

		var alertCount int
		var alertMu sync.Mutex

		manager.SetBudget(scope, &llm_dto.BudgetConfig{
			MaxDailySpend:  maths.NewMoneyFromString("100.00", testCurrency),
			AlertThreshold: 0.8,
			OnAlert: func(status llm_dto.BudgetStatus) {
				alertMu.Lock()
				alertCount++
				alertMu.Unlock()
			},
		})

		cost1 := &llm_dto.CostEstimate{
			TotalTokens: 100,
			TotalCost:   maths.NewMoneyFromString("85.00", testCurrency),
		}
		_ = manager.RecordUsage(ctx, scope, cost1)

		time.Sleep(50 * time.Millisecond)

		cost2 := &llm_dto.CostEstimate{
			TotalTokens: 50,
			TotalCost:   maths.NewMoneyFromString("10.00", testCurrency),
		}
		_ = manager.RecordUsage(ctx, scope, cost2)

		time.Sleep(50 * time.Millisecond)

		alertMu.Lock()
		assert.Equal(t, 1, alertCount, "alert should only be triggered once")
		alertMu.Unlock()
	})
}

func TestBudgetManager_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)
	scope := "test-scope"

	manager.SetBudget(scope, &llm_dto.BudgetConfig{
		MaxDailySpend: maths.NewMoneyFromString("10000.00", testCurrency),
	})

	done := make(chan struct{})

	go func() {
		for range 50 {
			_ = manager.CheckBudget(ctx, scope, maths.NewMoneyFromString("1.00", testCurrency))
		}
		done <- struct{}{}
	}()

	go func() {
		for range 50 {
			_ = manager.RecordUsage(ctx, scope, &llm_dto.CostEstimate{
				TotalTokens: 10,
				TotalCost:   maths.NewMoneyFromString("0.10", testCurrency),
			})
		}
		done <- struct{}{}
	}()

	go func() {
		for range 50 {
			_, _ = manager.GetStatus(ctx, scope)
		}
		done <- struct{}{}
	}()

	for range 3 {
		<-done
	}
}

func TestBudgetManager_GetStatus_RemainingBudgetMinOfAllLimits(t *testing.T) {
	ctx := context.Background()
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)
	scope := "test-scope"

	manager.SetBudget(scope, &llm_dto.BudgetConfig{
		MaxTotalSpend:  maths.NewMoneyFromString("1000.00", testCurrency),
		MaxDailySpend:  maths.NewMoneyFromString("100.00", testCurrency),
		MaxHourlySpend: maths.NewMoneyFromString("20.00", testCurrency),
	})

	store.mu.Lock()
	store.statuses[scope] = &llm_dto.BudgetStatus{
		Scope:       scope,
		TotalSpent:  maths.NewMoneyFromString("900.00", testCurrency),
		DailySpent:  maths.NewMoneyFromString("50.00", testCurrency),
		HourlySpent: maths.NewMoneyFromString("15.00", testCurrency),
	}
	store.mu.Unlock()

	status, err := manager.GetStatus(ctx, scope)
	require.NoError(t, err)

	assert.True(t, status.RemainingBudget.MustEquals(maths.NewMoneyFromString("5.00", testCurrency)),
		"remaining budget should be min(100-50=50, 20-15=5, 1000-900=100) = 5.00, got %s", status.RemainingBudget.MustString())
}

func TestBudgetManager_AlertCallback_PanicRecovery(t *testing.T) {
	ctx := context.Background()
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)
	scope := "test-scope"

	manager.SetBudget(scope, &llm_dto.BudgetConfig{
		MaxDailySpend:  maths.NewMoneyFromString("100.00", testCurrency),
		AlertThreshold: 0.8,
		OnAlert: func(_ llm_dto.BudgetStatus) {
			panic("callback panic")
		},
	})

	cost := &llm_dto.CostEstimate{
		TotalTokens: 100,
		TotalCost:   maths.NewMoneyFromString("85.00", testCurrency),
	}

	err := manager.RecordUsage(ctx, scope, cost)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
}

func TestBudgetManager_UnreserveBudget(t *testing.T) {
	ctx := context.Background()
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)
	scope := "test-scope"

	var unreservedScope string
	var unreservedCost maths.Money
	store.UnreserveFunc = func(_ context.Context, s string, c maths.Money) error {
		unreservedScope = s
		unreservedCost = c
		return nil
	}

	cost := maths.NewMoneyFromString("5.00", testCurrency)
	err := manager.UnreserveBudget(ctx, scope, cost)

	require.NoError(t, err)
	assert.Equal(t, scope, unreservedScope)
	assert.True(t, unreservedCost.MustEquals(cost))
}

func TestBudgetManager_CheckBudget_AtomicCheckAndReserve(t *testing.T) {
	ctx := context.Background()
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)
	scope := "test-scope"

	manager.SetBudget(scope, &llm_dto.BudgetConfig{
		MaxDailySpend: maths.NewMoneyFromString("100.00", testCurrency),
	})

	var checkAndReserveCalled bool
	store.CheckAndReserveFunc = func(_ context.Context, s string, cost maths.Money, limits llm_dto.BudgetLimits) error {
		checkAndReserveCalled = true
		assert.Equal(t, scope, s)
		assert.True(t, limits.MaxDailySpend.MustEquals(maths.NewMoneyFromString("100.00", testCurrency)))
		return nil
	}

	err := manager.CheckBudget(ctx, scope, maths.NewMoneyFromString("10.00", testCurrency))

	require.NoError(t, err)
	assert.True(t, checkAndReserveCalled)
}

func TestBudgetManager_RecordUsage_StoreRecordError(t *testing.T) {
	ctx := context.Background()
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)

	recordErr := errors.New("storage unavailable")
	store.RecordFunc = func(_ context.Context, _ string, _ *llm_dto.CostEstimate) error {
		return recordErr
	}

	cost := &llm_dto.CostEstimate{
		InputTokens:  50,
		OutputTokens: 25,
		TotalTokens:  75,
		TotalCost:    maths.NewMoneyFromString("0.25", testCurrency),
	}

	err := manager.RecordUsage(ctx, "test-scope", cost)

	require.Error(t, err)
	assert.ErrorIs(t, err, recordErr)
}

func TestBudgetManager_RecordUsage_IncrementTokensError(t *testing.T) {
	ctx := context.Background()
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)

	tokenErr := errors.New("token increment failed")
	store.IncrementTokensFunc = func(_ context.Context, _ string, _ int64) error {
		return tokenErr
	}

	cost := &llm_dto.CostEstimate{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
		TotalCost:    maths.NewMoneyFromString("0.50", testCurrency),
	}

	err := manager.RecordUsage(ctx, "test-scope", cost)

	require.Error(t, err)
	assert.ErrorIs(t, err, tokenErr)
}
