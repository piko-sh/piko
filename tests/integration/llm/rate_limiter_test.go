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

//go:build integration

package llm_integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_adapters"
)

func TestRateLimiter_ExceedsLimit(t *testing.T) {
	service, ctx := createZoltaiService(t, llm_domain.WithRateLimiter(
		llm_domain.NewRateLimiter(ratelimiter_adapters.NewInMemoryTokenBucketStore()),
	))
	service.SetRateLimits("test", 1, 0)

	_, err := service.NewCompletion().
		User("First request").
		BudgetScope("test").
		Do(ctx)
	require.NoError(t, err)

	_, err = service.NewCompletion().
		User("Second request should be rate limited").
		BudgetScope("test").
		Do(ctx)

	assert.ErrorIs(t, err, llm_domain.ErrRateLimited)
}

func TestRateLimiter_WithinLimit(t *testing.T) {
	service, ctx := createZoltaiService(t, llm_domain.WithRateLimiter(
		llm_domain.NewRateLimiter(ratelimiter_adapters.NewInMemoryTokenBucketStore()),
	))
	service.SetRateLimits("test", 100, 0)

	for range 5 {
		response, err := service.NewCompletion().
			User("Request within limit").
			BudgetScope("test").
			Do(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, response.Content())
	}
}
