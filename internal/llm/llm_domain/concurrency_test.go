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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/maths"
)

func TestService_ConcurrentComplete(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	err := service.RegisterProvider(context.Background(), "mock", provider)
	require.NoError(t, err)

	const goroutines = 100
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errs := make([]error, goroutines)

	for i := range goroutines {
		go func(index int) {
			defer wg.Done()
			request := &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: fmt.Sprintf("Hello %d", index)},
				},
			}
			response, err := service.Complete(ctx, request)
			errs[index] = err
			if err == nil {
				assert.NotNil(t, response)
			}
		}(i)
	}

	wg.Wait()

	for i, e := range errs {
		assert.NoError(t, e, "goroutine %d returned an error", i)
	}

	callCount := atomic.LoadInt64(&provider.CompleteCallCount)
	assert.Equal(t, int64(goroutines), callCount)
}

func TestService_ConcurrentStream(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	provider.SupportsStreamingValue = true
	err := service.RegisterProvider(context.Background(), "mock", provider)
	require.NoError(t, err)

	const goroutines = 100
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errs := make([]error, goroutines)

	for i := range goroutines {
		go func(index int) {
			defer wg.Done()
			request := &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: fmt.Sprintf("Stream %d", index)},
				},
			}
			events, err := service.Stream(ctx, request)
			errs[index] = err
			if err == nil {

				for range events {
				}
			}
		}(i)
	}

	wg.Wait()

	for i, e := range errs {
		assert.NoError(t, e, "goroutine %d returned an error", i)
	}

	callCount := atomic.LoadInt64(&provider.StreamCallCount)
	assert.Equal(t, int64(goroutines), callCount)
}

func TestService_ConcurrentProviderRegistrationAndUsage(t *testing.T) {
	service := NewService("base")
	baseProvider := NewMockLLMProvider()
	err := service.RegisterProvider(context.Background(), "base", baseProvider)
	require.NoError(t, err)

	ctx := context.Background()
	const registerers = 20
	const users = 50
	var wg sync.WaitGroup
	wg.Add(registerers + users)

	for i := range registerers {
		go func(index int) {
			defer wg.Done()
			name := fmt.Sprintf("provider-%d", index)
			p := NewMockLLMProvider()

			_ = service.RegisterProvider(ctx, name, p)
		}(i)
	}

	for i := range users {
		go func(index int) {
			defer wg.Done()
			request := &llm_dto.CompletionRequest{
				Model: "gpt-4o",
				Messages: []llm_dto.Message{
					{Role: llm_dto.RoleUser, Content: fmt.Sprintf("Usage %d", index)},
				},
			}
			_, _ = service.Complete(ctx, request)
		}(i)
	}

	wg.Wait()

	assert.True(t, service.HasProvider("base"))
	assert.GreaterOrEqual(t, len(service.GetProviders()), 1)
}

func TestService_ConcurrentGetDefaultProvider(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	err := service.RegisterProvider(context.Background(), "mock", provider)
	require.NoError(t, err)

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make([]string, goroutines)

	for i := range goroutines {
		go func(index int) {
			defer wg.Done()
			results[index] = service.GetDefaultProvider()
		}(i)
	}

	wg.Wait()

	for i, r := range results {
		assert.Equal(t, "mock", r, "goroutine %d got unexpected default provider", i)
	}
}

func TestBudgetManager_ConcurrentCheckAndRecord(t *testing.T) {
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)
	scope := "concurrent-scope"

	manager.SetBudget(scope, &llm_dto.BudgetConfig{
		MaxDailySpend: maths.NewMoneyFromString("100000.00", testCurrency),
	})

	const goroutines = 100
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(index int) {
			defer wg.Done()
			if index%2 == 0 {
				_ = manager.CheckBudget(ctx, scope, maths.NewMoneyFromString("1.00", testCurrency))
			} else {
				_ = manager.RecordUsage(ctx, scope, &llm_dto.CostEstimate{
					TotalTokens: 10,
					TotalCost:   maths.NewMoneyFromString("0.10", testCurrency),
				})
			}
		}(i)
	}

	wg.Wait()

	status, err := manager.GetStatus(ctx, scope)
	require.NoError(t, err)
	assert.Equal(t, scope, status.Scope)
}

func TestBudgetManager_ConcurrentSetAndCheck(t *testing.T) {
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)

	const goroutines = 100
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(index int) {
			defer wg.Done()
			scope := fmt.Sprintf("scope-%d", index%10)
			if index%2 == 0 {
				manager.SetBudget(scope, &llm_dto.BudgetConfig{
					MaxDailySpend: maths.NewMoneyFromString("500.00", testCurrency),
				})
			} else {
				_ = manager.CheckBudget(ctx, scope, maths.NewMoneyFromString("1.00", testCurrency))
			}
		}(i)
	}

	wg.Wait()

	hasBudget := false
	for i := range 10 {
		if manager.HasBudget(fmt.Sprintf("scope-%d", i)) {
			hasBudget = true
			break
		}
	}
	assert.True(t, hasBudget, "at least one scope should have a budget configured")
}

func TestBudgetManager_ConcurrentResetAndRecord(t *testing.T) {
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)
	scope := "reset-scope"

	manager.SetBudget(scope, &llm_dto.BudgetConfig{
		MaxDailySpend: maths.NewMoneyFromString("100000.00", testCurrency),
	})

	const goroutines = 100
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(index int) {
			defer wg.Done()
			if index%3 == 0 {
				_ = manager.Reset(ctx, scope)
			} else {
				_ = manager.RecordUsage(ctx, scope, &llm_dto.CostEstimate{
					TotalTokens: 5,
					TotalCost:   maths.NewMoneyFromString("0.05", testCurrency),
				})
			}
		}(i)
	}

	wg.Wait()

	_, err := manager.GetStatus(ctx, scope)
	assert.NoError(t, err)
}

func TestBudgetManager_ConcurrentAlertThreshold(t *testing.T) {
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)
	scope := "alert-scope"

	var alertCount atomic.Int64

	manager.SetBudget(scope, &llm_dto.BudgetConfig{
		MaxDailySpend:  maths.NewMoneyFromString("100.00", testCurrency),
		AlertThreshold: 0.5,
		OnAlert: func(_ llm_dto.BudgetStatus) {
			alertCount.Add(1)
		},
	})

	const goroutines = 100
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = manager.RecordUsage(ctx, scope, &llm_dto.CostEstimate{
				TotalTokens: 10,
				TotalCost:   maths.NewMoneyFromString("1.00", testCurrency),
			})
		}()
	}

	wg.Wait()

	time.Sleep(100 * time.Millisecond)

	count := alertCount.Load()
	assert.GreaterOrEqual(t, count, int64(1),
		"alert should fire at least once since threshold is exceeded")
}

func TestBudgetManager_ConcurrentResetDaily(t *testing.T) {
	store := NewMockBudgetStore()
	calc := NewCostCalculator()
	manager := NewBudgetManager(store, calc)

	for i := range 5 {
		scope := fmt.Sprintf("daily-scope-%d", i)
		manager.SetBudget(scope, &llm_dto.BudgetConfig{
			MaxDailySpend: maths.NewMoneyFromString("1000.00", testCurrency),
		})
	}

	const goroutines = 100
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(index int) {
			defer wg.Done()
			if index%5 == 0 {
				_ = manager.ResetDaily(ctx)
			} else {
				scope := fmt.Sprintf("daily-scope-%d", index%5)
				_ = manager.RecordUsage(ctx, scope, &llm_dto.CostEstimate{
					TotalTokens: 10,
					TotalCost:   maths.NewMoneyFromString("0.50", testCurrency),
				})
			}
		}(i)
	}

	wg.Wait()

	for i := range 5 {
		scope := fmt.Sprintf("daily-scope-%d", i)
		_, err := manager.GetStatus(ctx, scope)
		assert.NoError(t, err, "GetStatus should succeed for %s", scope)
	}
}
