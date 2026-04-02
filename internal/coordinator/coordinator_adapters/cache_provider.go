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

package coordinator_adapters

import (
	"fmt"

	cache_adapters_otter "piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
)

const (
	// BlueprintBuildResults is the factory blueprint name for creating typed
	// build result caches via the cache hexagon.
	BlueprintBuildResults = "coordinator-build-results"

	// BlueprintIntrospection is the factory blueprint name for creating typed
	// introspection caches via the cache hexagon.
	BlueprintIntrospection = "coordinator-introspection"
)

// createBuildResultCache is the factory function for creating typed build
// result caches. It type-asserts the options and delegates to the Otter
// provider factory.
//
// Takes options (any) which must be
// cache_dto.Options[string, *annotator_dto.ProjectAnnotationResult].
//
// Returns any which is the created typed cache instance.
// Returns error when the options type is incorrect or cache creation
// fails.
func createBuildResultCache(
	_ cache_domain.Service,
	_ string,
	options any,
) (any, error) {
	opts, ok := options.(cache_dto.Options[string, *annotator_dto.ProjectAnnotationResult])
	if !ok {
		return nil, fmt.Errorf(
			"invalid options type for build result cache: expected cache_dto.Options[string, *annotator_dto.ProjectAnnotationResult], got %T",
			options,
		)
	}

	cache, err := cache_adapters_otter.OtterProviderFactory[string, *annotator_dto.ProjectAnnotationResult](opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create build result cache: %w", err)
	}

	return cache, nil
}

// createIntrospectionCache is the factory function for creating typed
// introspection caches. It type-asserts the options and delegates to the Otter
// provider factory.
//
// Takes options (any) which must be
// cache_dto.Options[string, *coordinator_domain.IntrospectionCacheEntry].
//
// Returns any which is the created typed cache instance.
// Returns error when the options type is incorrect or cache creation
// fails.
func createIntrospectionCache(
	_ cache_domain.Service,
	_ string,
	options any,
) (any, error) {
	opts, ok := options.(cache_dto.Options[string, *coordinator_domain.IntrospectionCacheEntry])
	if !ok {
		return nil, fmt.Errorf(
			"invalid options type for introspection cache: expected cache_dto.Options[string, *coordinator_domain.IntrospectionCacheEntry], got %T",
			options,
		)
	}

	cache, err := cache_adapters_otter.OtterProviderFactory[string, *coordinator_domain.IntrospectionCacheEntry](opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create introspection cache: %w", err)
	}

	return cache, nil
}

func init() {
	cache_domain.RegisterProviderFactory(
		BlueprintBuildResults,
		createBuildResultCache,
	)
	cache_domain.RegisterProviderFactory(
		BlueprintIntrospection,
		createIntrospectionCache,
	)
}
