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

package llm_test

import (
	"context"
	"errors"
	"testing"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/llm/llm_adapters/vector_cache"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func createOtterVectorStore(t *testing.T, dimension int, metric string) *vector_cache.Store {
	t.Helper()

	bp := "otter-vec-" + t.Name()
	cache_domain.UnregisterProviderFactory(bp)
	cache_domain.RegisterProviderFactory(bp, func(_ cache_domain.Service, _ string, options any) (any, error) {
		opts, ok := options.(cache_dto.Options[string, llm_dto.VectorDocument])
		if !ok {
			return nil, errors.New("invalid options type")
		}
		return provider_otter.OtterProviderFactory[string, llm_dto.VectorDocument](opts)
	})
	t.Cleanup(func() { cache_domain.UnregisterProviderFactory(bp) })

	cacheService := cache_domain.NewService("otter")

	return vector_cache.New(func(ns string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		m := metric
		if config != nil && config.Metric != "" {
			m = string(config.Metric)
		}
		d := dimension
		if config != nil && config.Dimension > 0 {
			d = config.Dimension
		}

		schema := cache_dto.NewSearchSchema(
			cache_dto.VectorFieldWithMetric("Vector", d, m),
			cache_dto.TextField("Content"),
		)

		return cache_domain.NewCacheBuilder[string, llm_dto.VectorDocument](cacheService).
			FactoryBlueprint(bp).
			Namespace(ns).
			MaximumSize(100000).
			Searchable(schema).
			Build(context.Background())
	})
}
