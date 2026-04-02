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

package conformance

import (
	"context"
	"testing"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
)

func OtterStringFactory(t *testing.T, opts cache_dto.Options[string, string]) cache_domain.Cache[string, string] {
	t.Helper()

	cache, err := provider_otter.OtterProviderFactory(opts)
	if err != nil {
		t.Fatalf("failed to create otter cache: %v", err)
	}

	t.Cleanup(func() {
		_ = cache.Close(context.Background())
	})

	return cache
}

func OtterProductFactory(t *testing.T, opts cache_dto.Options[string, Product]) cache_domain.Cache[string, Product] {
	t.Helper()

	cache, err := provider_otter.OtterProviderFactory(opts)
	if err != nil {
		t.Fatalf("failed to create otter cache: %v", err)
	}

	t.Cleanup(func() {
		_ = cache.Close(context.Background())
	})

	return cache
}

func TestOtterConformance(t *testing.T) {
	t.Parallel()

	config := StringConfig{
		ProviderFactory:      OtterStringFactory,
		SupportsSearch:       true,
		SupportsTTL:          true,
		SupportsIteration:    true,
		SupportsCompute:      true,
		SupportsMaximum:      true,
		SupportsWeightedSize: true,
		SupportsRefresh:      true,
	}

	RunStringSuite(t, config)
}

func TestOtterSearchConformance(t *testing.T) {
	t.Parallel()

	config := ProductConfig{
		ProviderFactory: OtterProductFactory,
	}

	RunSearchSuite(t, config)
}

func NewOtterStringConfig() StringConfig {
	return StringConfig{
		ProviderFactory:      OtterStringFactory,
		SupportsSearch:       true,
		SupportsTTL:          true,
		SupportsIteration:    true,
		SupportsCompute:      true,
		SupportsMaximum:      true,
		SupportsWeightedSize: true,
		SupportsRefresh:      true,
	}
}

func NewOtterProductConfig() ProductConfig {
	return ProductConfig{
		ProviderFactory: OtterProductFactory,
	}
}
