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

package lifecycle_domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestExternalComponentResolution_SkippedInProductionMode(t *testing.T) {
	t.Parallel()

	var resolverCalled bool
	resolver := &resolver_domain.MockResolver{
		FindModuleBoundaryFunc: func(_ context.Context, _ string) (string, string, error) {
			resolverCalled = true
			return "", "", errors.New("resolver must not be called in production mode")
		},
	}
	ls := &lifecycleService{
		productionMode: true,
		resolver:       resolver,
		externalComponents: []component_dto.ComponentDefinition{
			{ModulePath: "piko.sh/piko/components", AssetPaths: []string{"icons"}},
		},
	}

	assert.Nil(t, ls.resolveExternalComponentDirs(t.Context()),
		"production mode must skip external component directory resolution")
	assert.Nil(t, ls.collectExternalAssetPairs(t.Context()),
		"production mode must skip external asset pair collection")
	assert.False(t, resolverCalled,
		"production mode must not resolve external component module boundaries (the source is already baked in)")
}
