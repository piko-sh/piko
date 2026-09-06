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

package bootstrap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/clock"
)

func TestContainer_NewLifecycleServiceDeps(t *testing.T) {
	registryService := &registry_domain.MockRegistryService{}
	coordinatorService := &coordinator_domain.MockCoordinatorService{}
	resolver := &resolver_domain.MockResolver{}
	mockClock := clock.NewMockClock(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))

	testCases := []struct {
		configured clock.Clock
		name       string
		wantMock   bool
	}{
		{name: "keeps a configured clock", configured: mockClock, wantMock: true},
		{name: "falls back to the real clock", configured: nil, wantMock: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewContainer()
			c.renderRegistryOverride = &stubRenderRegistryPort{}
			c.rendererOverride = &stubRenderService{}
			serviceConfig := &lifecycleServiceConfig{}
			serviceConfig.Clock = tc.configured
			serviceConfig.PathsConfig = lifecycle_domain.LifecyclePathsConfig{}

			deps := c.newLifecycleServiceDeps(serviceConfig, registryService, coordinatorService, resolver)

			require.NotNil(t, deps)
			assert.Same(t, registryService, deps.RegistryService)
			assert.Same(t, coordinatorService, deps.CoordinatorService)
			assert.Same(t, resolver, deps.Resolver)
			assert.Same(t, c.renderRegistryOverride, deps.RenderRegistryPort)
			assert.Same(t, c.rendererOverride, deps.Renderer)
			assert.Nil(t, deps.CaptchaService, "the captcha service is attached separately")
			assert.Equal(t, c.isProductionMode(), deps.ProductionMode)
			assert.Equal(t, c.registryBlobsReadOnly(), deps.RegistryBlobsReadOnly)
			require.NotNil(t, deps.Clock)
			_, isMock := deps.Clock.(*clock.MockClock)
			assert.Equal(t, tc.wantMock, isMock)
		})
	}
}
