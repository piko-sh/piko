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

package orchestrator_adapters

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestNewArtefactWorkflowBridge(t *testing.T) {
	t.Parallel()

	bridge := NewArtefactWorkflowBridge(nil, nil, nil, nil)
	require.NotNil(t, bridge)
	assert.Nil(t, bridge.registryService)
	assert.Nil(t, bridge.orchestratorService)
	assert.Nil(t, bridge.taskDispatcher)
	assert.Nil(t, bridge.eventBus)
	assert.Equal(t, int64(0), bridge.ArtefactEventsProcessed())
}

func TestArtefactEventsProcessed(t *testing.T) {
	t.Parallel()

	bridge := NewArtefactWorkflowBridge(nil, nil, nil, nil)
	assert.Equal(t, int64(0), bridge.ArtefactEventsProcessed())

	bridge.artefactEventsProcessed.Add(5)
	assert.Equal(t, int64(5), bridge.ArtefactEventsProcessed())

	bridge.artefactEventsProcessed.Add(3)
	assert.Equal(t, int64(8), bridge.ArtefactEventsProcessed())
}

func TestBuildVariantStatusMap_Comprehensive(t *testing.T) {
	t.Parallel()

	t.Run("single ready variant", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			{VariantID: "thumb", Status: registry_dto.VariantStatusReady},
		}
		m := buildVariantStatusMap(variants)
		assert.Len(t, m, 1)
		assert.Equal(t, registry_dto.VariantStatusReady, m["thumb"])
	})

	t.Run("multiple variants with different statuses", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			{VariantID: "source", Status: registry_dto.VariantStatusReady},
			{VariantID: "thumb", Status: registry_dto.VariantStatusPending},
			{VariantID: "webp", Status: registry_dto.VariantStatusStale},
		}
		m := buildVariantStatusMap(variants)
		assert.Len(t, m, 3)
		assert.Equal(t, registry_dto.VariantStatusReady, m["source"])
		assert.Equal(t, registry_dto.VariantStatusPending, m["thumb"])
		assert.Equal(t, registry_dto.VariantStatusStale, m["webp"])
	})

	t.Run("duplicate variant IDs last one wins", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			{VariantID: "thumb", Status: registry_dto.VariantStatusPending},
			{VariantID: "thumb", Status: registry_dto.VariantStatusReady},
		}
		m := buildVariantStatusMap(variants)
		assert.Len(t, m, 1)
		assert.Equal(t, registry_dto.VariantStatusReady, m["thumb"])
	})
}

func TestIsProfileAlreadyReady_Comprehensive(t *testing.T) {
	t.Parallel()

	variantStatus := map[string]registry_dto.VariantStatus{
		"thumb": registry_dto.VariantStatusReady,
		"webp":  registry_dto.VariantStatusPending,
		"stale": registry_dto.VariantStatusStale,
	}

	tests := []struct {
		name     string
		profile  string
		expected bool
	}{
		{name: "ready variant is ready", profile: "thumb", expected: true},
		{name: "pending variant is not ready", profile: "webp", expected: false},
		{name: "stale variant is not ready", profile: "stale", expected: false},
		{name: "missing variant is not ready", profile: "missing", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isProfileAlreadyReady(variantStatus, tc.profile))
		})
	}
}

func TestFindMissingDependencies(t *testing.T) {
	t.Parallel()

	t.Run("no dependencies returns empty", func(t *testing.T) {
		t.Parallel()
		deps := &registry_dto.Dependencies{}
		variantStatus := map[string]registry_dto.VariantStatus{}
		missing := findMissingDependencies(deps, variantStatus)
		assert.Empty(t, missing)
	})

	t.Run("all dependencies ready returns empty", func(t *testing.T) {
		t.Parallel()
		deps := &registry_dto.Dependencies{}
		deps.Add("source")
		deps.Add("thumb")
		variantStatus := map[string]registry_dto.VariantStatus{
			"source": registry_dto.VariantStatusReady,
			"thumb":  registry_dto.VariantStatusReady,
		}
		missing := findMissingDependencies(deps, variantStatus)
		assert.Empty(t, missing)
	})

	t.Run("missing dependency found", func(t *testing.T) {
		t.Parallel()
		deps := &registry_dto.Dependencies{}
		deps.Add("source")
		deps.Add("thumb")
		variantStatus := map[string]registry_dto.VariantStatus{
			"source": registry_dto.VariantStatusReady,
		}
		missing := findMissingDependencies(deps, variantStatus)
		assert.Contains(t, missing, "thumb")
		assert.Len(t, missing, 1)
	})

	t.Run("pending dependency is missing", func(t *testing.T) {
		t.Parallel()
		deps := &registry_dto.Dependencies{}
		deps.Add("source")
		variantStatus := map[string]registry_dto.VariantStatus{
			"source": registry_dto.VariantStatusPending,
		}
		missing := findMissingDependencies(deps, variantStatus)
		assert.Contains(t, missing, "source")
	})

	t.Run("stale dependency is missing", func(t *testing.T) {
		t.Parallel()
		deps := &registry_dto.Dependencies{}
		deps.Add("source")
		variantStatus := map[string]registry_dto.VariantStatus{
			"source": registry_dto.VariantStatusStale,
		}
		missing := findMissingDependencies(deps, variantStatus)
		assert.Contains(t, missing, "source")
	})

	t.Run("multiple missing dependencies", func(t *testing.T) {
		t.Parallel()
		deps := &registry_dto.Dependencies{}
		deps.Add("dep1")
		deps.Add("dep2")
		deps.Add("dep3")
		variantStatus := map[string]registry_dto.VariantStatus{
			"dep1": registry_dto.VariantStatusReady,
		}
		missing := findMissingDependencies(deps, variantStatus)
		assert.Len(t, missing, 2)
		assert.Contains(t, missing, "dep2")
		assert.Contains(t, missing, "dep3")
	})
}

func TestMapPriority(t *testing.T) {
	t.Parallel()

	bridge := &ArtefactWorkflowBridge{}

	tests := []struct {
		name     string
		input    registry_dto.ProfilePriority
		expected orchestrator_domain.TaskPriority
	}{
		{name: "NEED maps to PriorityHigh", input: registry_dto.PriorityNeed, expected: orchestrator_domain.PriorityHigh},
		{name: "WANT maps to PriorityLow", input: registry_dto.PriorityWant, expected: orchestrator_domain.PriorityLow},
		{name: "empty string defaults to PriorityNormal", input: "", expected: orchestrator_domain.PriorityNormal},
		{name: "unknown value defaults to PriorityNormal", input: "UNKNOWN", expected: orchestrator_domain.PriorityNormal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, bridge.mapPriority(tc.input))
		})
	}
}

func TestHandleEventWithAck_MissingArtefactID(t *testing.T) {
	t.Parallel()

	bridge := NewArtefactWorkflowBridge(nil, nil, nil, nil)
	ctx := t.Context()

	t.Run("missing artefactID acks message", func(t *testing.T) {
		t.Parallel()
		event := orchestrator_domain.Event{
			Type:    orchestrator_domain.EventType("artefact.created"),
			Payload: map[string]any{"other": "data"},
		}
		err := bridge.handleEventWithAck(ctx, event)
		assert.NoError(t, err, "should Ack to prevent redelivery")
	})

	t.Run("empty artefactID acks message", func(t *testing.T) {
		t.Parallel()
		event := orchestrator_domain.Event{
			Type:    orchestrator_domain.EventType("artefact.created"),
			Payload: map[string]any{"artefactID": ""},
		}
		err := bridge.handleEventWithAck(ctx, event)
		assert.NoError(t, err, "should Ack empty artefactID")
	})

	t.Run("non-string artefactID acks message", func(t *testing.T) {
		t.Parallel()
		event := orchestrator_domain.Event{
			Type:    orchestrator_domain.EventType("artefact.created"),
			Payload: map[string]any{"artefactID": 42},
		}
		err := bridge.handleEventWithAck(ctx, event)
		assert.NoError(t, err, "should Ack non-string artefactID")
	})

	t.Run("nil payload acks message", func(t *testing.T) {
		t.Parallel()
		event := orchestrator_domain.Event{
			Type:    orchestrator_domain.EventType("artefact.created"),
			Payload: nil,
		}
		err := bridge.handleEventWithAck(ctx, event)
		assert.NoError(t, err, "should Ack nil payload")
	})
}

func TestFinaliseEventHandling_Success(t *testing.T) {
	t.Parallel()

	bridge := NewArtefactWorkflowBridge(nil, nil, nil, nil)
	ctx := t.Context()

	_, span, _ := log.Span(ctx, "test")
	defer span.End()

	initialCount := bridge.ArtefactEventsProcessed()
	bridge.finaliseEventHandling(ctx, span, nil, "art-1", time.Now())
	assert.Equal(t, initialCount+1, bridge.ArtefactEventsProcessed())
}

func TestFinaliseEventHandling_Error(t *testing.T) {
	t.Parallel()

	bridge := NewArtefactWorkflowBridge(nil, nil, nil, nil)
	ctx := t.Context()

	_, span, _ := log.Span(ctx, "test")
	defer span.End()

	initialCount := bridge.ArtefactEventsProcessed()
	bridge.finaliseEventHandling(ctx, span, assert.AnError, "art-1", time.Now())
	assert.Equal(t, initialCount+1, bridge.ArtefactEventsProcessed())
}

func TestEvaluateAndDispatchProfiles_EmptyProfiles(t *testing.T) {
	t.Parallel()

	bridge := NewArtefactWorkflowBridge(nil, nil, nil, nil)
	ctx := t.Context()

	artefact := &registry_dto.ArtefactMeta{
		ID:              "art-1",
		DesiredProfiles: []registry_dto.NamedProfile{},
	}
	variantStatus := map[string]registry_dto.VariantStatus{}

	dispatched := bridge.evaluateAndDispatchProfiles(ctx, artefact, variantStatus)
	assert.Equal(t, 0, dispatched)
}
