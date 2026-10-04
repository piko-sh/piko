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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/analytics/analytics_dto"
)

type recordingAnalyticsCollector struct {
	name string
}

func (*recordingAnalyticsCollector) Start(context.Context) {}

func (*recordingAnalyticsCollector) Collect(context.Context, *analytics_dto.Event) error {
	return nil
}

func (*recordingAnalyticsCollector) Flush(context.Context) error { return nil }

func (*recordingAnalyticsCollector) Close(context.Context) error { return nil }

func (c *recordingAnalyticsCollector) Name() string { return c.name }

func TestGetAnalyticsService(t *testing.T) {
	testCases := []struct {
		name           string
		collectorNames []string
		wantService    bool
	}{
		{name: "no collectors leaves analytics disabled", collectorNames: nil, wantService: false},
		{name: "a registered collector creates the service", collectorNames: []string{"first"}, wantService: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			container := NewContainer()
			for _, collectorName := range testCase.collectorNames {
				container.AddAnalyticsCollector(&recordingAnalyticsCollector{name: collectorName})
			}

			service := container.GetAnalyticsService()

			if !testCase.wantService {
				assert.Nil(t, service)
				return
			}
			require.NotNil(t, service)
			t.Cleanup(func() { require.NoError(t, service.Close(context.Background())) })
			assert.Same(t, service, container.GetAnalyticsService(), "the service is created once per container")
		})
	}
}

func TestGetAnalyticsService_EachContainerOwnsItsService(t *testing.T) {
	firstContainer := NewContainer()
	firstContainer.AddAnalyticsCollector(&recordingAnalyticsCollector{name: "first"})
	secondContainer := NewContainer()
	secondContainer.AddAnalyticsCollector(&recordingAnalyticsCollector{name: "second"})

	firstService := firstContainer.GetAnalyticsService()
	require.NotNil(t, firstService)
	t.Cleanup(func() { require.NoError(t, firstService.Close(context.Background())) })
	assert.Same(t, firstService, GetGlobalAnalyticsService())

	secondService := secondContainer.GetAnalyticsService()
	require.NotNil(t, secondService)
	t.Cleanup(func() { require.NoError(t, secondService.Close(context.Background())) })

	assert.NotSame(t, firstService, secondService, "containers must not share an analytics service")
	assert.Same(t, secondService, GetGlobalAnalyticsService(), "the most recently started service is the active one")
}
