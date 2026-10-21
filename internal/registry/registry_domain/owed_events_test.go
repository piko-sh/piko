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

package registry_domain

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func TestPublishEvent_FailedPublishStaysOwed(t *testing.T) {
	t.Parallel()

	var refuse atomic.Bool
	refuse.Store(true)
	bus := &MockEventBus{PublishFunc: func(context.Context, string, orchestrator_domain.Event) error {
		if refuse.Load() {
			return errors.New("bus full")
		}
		return nil
	}}
	service, ok := NewRegistryService(nil, nil, bus, nil).(*registryService)
	require.True(t, ok, "NewRegistryService returns a *registryService")
	ctx := t.Context()

	service.publishEvent(ctx, EventArtefactCreated, "assets/a.svg")
	assert.Equal(t, int64(1), service.ArtefactEventsIssued(), "an event counts as issued before it is published")
	assert.Equal(t, 1, service.FlushArtefactEvents(ctx), "the failed event stays owed")

	service.publishEvent(ctx, EventArtefactCreated, "assets/a.svg")
	assert.Equal(t, int64(1), service.ArtefactEventsIssued(), "a newer event replaces an owed one without counting twice")

	refuse.Store(false)
	assert.Zero(t, service.FlushArtefactEvents(ctx), "the owed event is delivered once the bus takes it")
	assert.Equal(t, int64(1), service.ArtefactEventsIssued())
	assert.Equal(t, int64(1), service.ArtefactEventsPublished(), "the deprecated count matches")

	service.publishEvent(ctx, EventArtefactCreated, "assets/b.svg")
	assert.Equal(t, int64(2), service.ArtefactEventsIssued())
	assert.Zero(t, service.FlushArtefactEvents(ctx))
}
