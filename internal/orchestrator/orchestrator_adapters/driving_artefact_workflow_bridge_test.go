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
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
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

	t.Run("derived variants are judged against their parent's current content", func(t *testing.T) {
		t.Parallel()
		derived := func(id, parentID, parentHash string) registry_dto.Variant {
			return registry_dto.Variant{
				VariantID: id,
				Status:    registry_dto.VariantStatusReady,
				Kind:      registry_dto.KindDerived,
				Transform: registry_dto.VariantTransform{ParentVariantID: parentID, ParentContentHash: parentHash},
			}
		}
		variants := []registry_dto.Variant{
			{VariantID: "source", Status: registry_dto.VariantStatusReady, Kind: registry_dto.KindSource, ContentHash: "new"},
			derived("current", "source", "new"),
			derived("outdated", "source", "old"),
			derived("orphaned", "missing", "old"),
			{VariantID: "unstamped", Status: registry_dto.VariantStatusReady, Kind: registry_dto.KindDerived},
		}
		m := buildVariantStatusMap(variants)
		assert.Equal(t, registry_dto.VariantStatusReady, m["current"])
		assert.Equal(t, registry_dto.VariantStatusStale, m["outdated"],
			"a variant finished from the old source after the source changed must be rebuilt")
		assert.Equal(t, registry_dto.VariantStatusStale, m["orphaned"])
		assert.Equal(t, registry_dto.VariantStatusReady, m["unstamped"])
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

func TestProfileNeedsBuild(t *testing.T) {
	t.Parallel()

	thumbProfile := func(dependsOn ...string) registry_dto.NamedProfile {
		return registry_dto.NamedProfile{
			Name: "thumb",
			Profile: registry_dto.DesiredProfile{
				CapabilityName: "resize",
				DependsOn:      registry_dto.DependenciesFromSlice(dependsOn),
			},
		}
	}
	readySource := registry_dto.Variant{VariantID: "source", Status: registry_dto.VariantStatusReady}

	testCases := []struct {
		name     string
		artefact registry_dto.ArtefactMeta
		want     bool
	}{
		{
			name: "desired profile with ready dependencies and no variant needs building",
			artefact: registry_dto.ArtefactMeta{
				DesiredProfiles: []registry_dto.NamedProfile{thumbProfile("source")},
				ActualVariants:  []registry_dto.Variant{readySource},
			},
			want: true,
		},
		{
			name: "profile whose variant is ready is already built",
			artefact: registry_dto.ArtefactMeta{
				DesiredProfiles: []registry_dto.NamedProfile{thumbProfile("source")},
				ActualVariants: []registry_dto.Variant{
					readySource,
					{VariantID: "thumb", Status: registry_dto.VariantStatusReady},
				},
			},
			want: false,
		},
		{
			name: "profile whose variant is stale needs building again",
			artefact: registry_dto.ArtefactMeta{
				DesiredProfiles: []registry_dto.NamedProfile{thumbProfile("source")},
				ActualVariants: []registry_dto.Variant{
					readySource,
					{VariantID: "thumb", Status: registry_dto.VariantStatusStale},
				},
			},
			want: true,
		},
		{
			name: "profile with a missing dependency cannot be built yet",
			artefact: registry_dto.ArtefactMeta{
				DesiredProfiles: []registry_dto.NamedProfile{thumbProfile("source", "medium")},
				ActualVariants:  []registry_dto.Variant{readySource},
			},
			want: false,
		},
		{
			name: "profile no longer desired is not built",
			artefact: registry_dto.ArtefactMeta{
				ActualVariants: []registry_dto.Variant{readySource},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, profileNeedsBuild(&tc.artefact, "thumb"))
		})
	}
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

	dispatched, err := bridge.evaluateAndDispatchProfiles(ctx, artefact, variantStatus)
	assert.NoError(t, err)
	assert.Equal(t, 0, dispatched)
}

func TestHandleEvent_RetriesTransientFetchFailure(t *testing.T) {
	t.Parallel()

	var reads atomic.Int64
	registry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
			if reads.Add(1) < 3 {
				return nil, errors.New("registry briefly unavailable")
			}
			return &registry_dto.ArtefactMeta{ID: "art-1"}, nil
		},
	}
	bridge := NewArtefactWorkflowBridge(registry, nil, &mockTaskDispatcher{}, nil)

	bridge.handleEvent(t.Context(), orchestrator_domain.Event{
		Type:    registry_domain.EventArtefactCreated,
		Payload: map[string]any{logKeyArtefactID: "art-1"},
	})
	assert.Zero(t, bridge.ArtefactEventsProcessed(), "an event being retried is still outstanding")

	require.Eventually(t, func() bool { return bridge.ArtefactEventsProcessed() == 1 }, 5*time.Second, 5*time.Millisecond,
		"the event is processed once the artefact can be read")
	assert.Equal(t, int64(3), reads.Load())
}

func TestHandleEvent_DoesNotRetryDeletedArtefact(t *testing.T) {
	t.Parallel()

	var reads atomic.Int64
	registry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
			reads.Add(1)
			return nil, registry_domain.ErrArtefactNotFound
		},
	}
	bridge := NewArtefactWorkflowBridge(registry, nil, &mockTaskDispatcher{}, nil)

	bridge.handleEvent(t.Context(), orchestrator_domain.Event{
		Type:    registry_domain.EventArtefactCreated,
		Payload: map[string]any{logKeyArtefactID: "gone"},
	})
	assert.Equal(t, int64(1), bridge.ArtefactEventsProcessed())
	assert.Equal(t, int64(1), reads.Load(), "a deleted artefact cannot appear by retrying")
}

func TestReconcileArtefacts_ReadsTheRegistryInBatches(t *testing.T) {
	t.Parallel()

	ids := make([]string, 0, 2*reconcileBatchSize+10)
	for i := range cap(ids) {
		ids = append(ids, fmt.Sprintf("asset-%d.svg", i))
	}
	unbuilt := func(id string) *registry_dto.ArtefactMeta {
		return &registry_dto.ArtefactMeta{
			ID:              id,
			ActualVariants:  []registry_dto.Variant{{VariantID: "source", Status: registry_dto.VariantStatusReady}},
			DesiredProfiles: []registry_dto.NamedProfile{makeMinifyProfile("source")},
		}
	}
	var batches [][]string
	registry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, id string) (*registry_dto.ArtefactMeta, error) {
			return unbuilt(id), nil
		},
		GetMultipleArtefactsFunc: func(_ context.Context, batch []string) ([]*registry_dto.ArtefactMeta, error) {
			batches = append(batches, batch)
			if len(batches) == 2 {
				return nil, errors.New("registry briefly unavailable")
			}
			artefacts := make([]*registry_dto.ArtefactMeta, 0, len(batch))
			for _, id := range batch[:len(batch)-1] {
				artefacts = append(artefacts, unbuilt(id))
			}
			return artefacts, nil
		},
	}
	dispatcher := &mockTaskDispatcher{}
	bridge := NewArtefactWorkflowBridge(registry, nil, dispatcher, nil)

	dispatched, err := bridge.ReconcileArtefacts(t.Context(), ids)

	require.Error(t, err, "a batch that cannot be read is reported")
	assert.Len(t, batches, 3, "one registry query per batch of reconcileBatchSize")
	assert.Equal(t, []int{reconcileBatchSize, reconcileBatchSize, 10}, []int{len(batches[0]), len(batches[1]), len(batches[2])})
	assert.Equal(t, reconcileBatchSize-1+9, dispatched,
		"the other batches are still re-checked, and an artefact that no longer exists is skipped")
	assert.Len(t, dispatcher.dispatched, dispatched)
}

func makeMinifyProfile(dependsOn string) registry_dto.NamedProfile {
	return registry_dto.NamedProfile{
		Name: "minified",
		Profile: registry_dto.DesiredProfile{
			CapabilityName: "minify-svg",
			DependsOn:      registry_dto.DependenciesFromSlice([]string{dependsOn}),
		},
	}
}
