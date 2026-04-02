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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func drainReader(_ context.Context, _ string, data io.Reader) error {
	_, _ = io.ReadAll(data)
	return nil
}

type mockTaskDispatcher struct {
	dispatchFunction func(ctx context.Context, task *orchestrator_domain.Task) error
	dispatched       []*orchestrator_domain.Task
}

func (m *mockTaskDispatcher) Dispatch(ctx context.Context, task *orchestrator_domain.Task) error {
	if m.dispatchFunction != nil {
		return m.dispatchFunction(ctx, task)
	}
	m.dispatched = append(m.dispatched, task)
	return nil
}

func (m *mockTaskDispatcher) DispatchDelayed(_ context.Context, _ *orchestrator_domain.Task, _ time.Time) error {
	return nil
}

func (m *mockTaskDispatcher) RegisterExecutor(_ context.Context, _ string, _ orchestrator_domain.TaskExecutor) {
}

func (m *mockTaskDispatcher) Start(_ context.Context) error { return nil }

func (m *mockTaskDispatcher) Stats() orchestrator_domain.DispatcherStats {
	return orchestrator_domain.DispatcherStats{}
}

func (m *mockTaskDispatcher) IsIdle() bool { return true }
func (m *mockTaskDispatcher) FailedTasks(_ context.Context) ([]orchestrator_domain.FailedTaskSummary, error) {
	return nil, nil
}
func (m *mockTaskDispatcher) SetBuildTag(_ string) {}
func (m *mockTaskDispatcher) BuildTag() string     { return "" }

func TestHandleEvent_MissingArtefactID(t *testing.T) {
	t.Parallel()

	bridge := NewArtefactWorkflowBridge(nil, nil, nil, nil)
	ctx := t.Context()

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("artefact.created"),
		Payload: map[string]any{},
	}

	initialCount := bridge.ArtefactEventsProcessed()
	bridge.handleEvent(ctx, event)

	assert.Equal(t, initialCount, bridge.ArtefactEventsProcessed())
}

func TestHandleEvent_DeletedArtefact(t *testing.T) {
	t.Parallel()

	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("artefact not found")
		},
	}
	bridge := NewArtefactWorkflowBridge(reg, nil, nil, nil)
	ctx := t.Context()

	event := orchestrator_domain.Event{
		Type:    registry_domain.EventArtefactDeleted,
		Payload: map[string]any{"artefactID": "art-deleted"},
	}

	initialCount := bridge.ArtefactEventsProcessed()
	bridge.handleEvent(ctx, event)
	assert.Equal(t, initialCount+1, bridge.ArtefactEventsProcessed())
}

func TestHandleEvent_FetchArtefactFails(t *testing.T) {
	t.Parallel()

	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("registry unavailable")
		},
	}
	bridge := NewArtefactWorkflowBridge(reg, nil, nil, nil)
	ctx := t.Context()

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("artefact.created"),
		Payload: map[string]any{"artefactID": "art-fail"},
	}

	initialCount := bridge.ArtefactEventsProcessed()
	bridge.handleEvent(ctx, event)
	assert.Equal(t, initialCount+1, bridge.ArtefactEventsProcessed())
}

func TestHandleEvent_ProfileAlreadyReady(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "art-1",
				DesiredProfiles: []registry_dto.NamedProfile{
					{
						Name: "thumb",
						Profile: registry_dto.DesiredProfile{
							CapabilityName: "resize",
							DependsOn:      deps,
						},
					},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
					{VariantID: "thumb", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
	}

	dispatcher := &mockTaskDispatcher{}
	bridge := NewArtefactWorkflowBridge(reg, nil, dispatcher, nil)
	ctx := t.Context()

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("artefact.created"),
		Payload: map[string]any{"artefactID": "art-1"},
	}

	bridge.handleEvent(ctx, event)
	assert.Empty(t, dispatcher.dispatched, "should not dispatch for already-ready profile")
}

func TestHandleEvent_DependenciesNotMet(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source", "medium"})
	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "art-1",
				DesiredProfiles: []registry_dto.NamedProfile{
					{
						Name: "thumb",
						Profile: registry_dto.DesiredProfile{
							CapabilityName: "resize",
							DependsOn:      deps,
						},
					},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
	}

	dispatcher := &mockTaskDispatcher{}
	bridge := NewArtefactWorkflowBridge(reg, nil, dispatcher, nil)
	ctx := t.Context()

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("artefact.created"),
		Payload: map[string]any{"artefactID": "art-1"},
	}

	bridge.handleEvent(ctx, event)
	assert.Empty(t, dispatcher.dispatched, "should not dispatch when dependencies not met")
}

func TestHandleEvent_DispatchesTask(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "art-1",
				DesiredProfiles: []registry_dto.NamedProfile{
					{
						Name: "thumb",
						Profile: registry_dto.DesiredProfile{
							CapabilityName: "resize",
							Priority:       registry_dto.PriorityNeed,
							DependsOn:      deps,
						},
					},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
	}

	dispatcher := &mockTaskDispatcher{}
	bridge := NewArtefactWorkflowBridge(reg, nil, dispatcher, nil)
	ctx := t.Context()

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("artefact.created"),
		Payload: map[string]any{"artefactID": "art-1"},
	}

	bridge.handleEvent(ctx, event)
	require.Len(t, dispatcher.dispatched, 1)
	assert.Equal(t, "artefact.compiler", dispatcher.dispatched[0].Executor)
	assert.Equal(t, "art-1:thumb", dispatcher.dispatched[0].DeduplicationKey)
	assert.Equal(t, orchestrator_domain.PriorityHigh, dispatcher.dispatched[0].Config.Priority)
}

func TestHandleEvent_DispatchReturnsDedup(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "art-1",
				DesiredProfiles: []registry_dto.NamedProfile{
					{
						Name: "thumb",
						Profile: registry_dto.DesiredProfile{
							CapabilityName: "resize",
							DependsOn:      deps,
						},
					},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
	}

	dispatcher := &mockTaskDispatcher{
		dispatchFunction: func(_ context.Context, _ *orchestrator_domain.Task) error {
			return orchestrator_domain.ErrDuplicateTask
		},
	}
	bridge := NewArtefactWorkflowBridge(reg, nil, dispatcher, nil)
	ctx := t.Context()

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("artefact.created"),
		Payload: map[string]any{"artefactID": "art-1"},
	}

	bridge.handleEvent(ctx, event)
}

func TestHandleEvent_DispatchFails(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "art-1",
				DesiredProfiles: []registry_dto.NamedProfile{
					{
						Name: "thumb",
						Profile: registry_dto.DesiredProfile{
							CapabilityName: "resize",
							DependsOn:      deps,
						},
					},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
	}

	dispatcher := &mockTaskDispatcher{
		dispatchFunction: func(_ context.Context, _ *orchestrator_domain.Task) error {
			return errors.New("dispatch failed")
		},
	}
	bridge := NewArtefactWorkflowBridge(reg, nil, dispatcher, nil)
	ctx := t.Context()

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("artefact.created"),
		Payload: map[string]any{"artefactID": "art-1"},
	}

	bridge.handleEvent(ctx, event)
}

func TestHandleEvent_MultipleProfiles(t *testing.T) {
	t.Parallel()

	sourceDeps := registry_dto.DependenciesFromSlice([]string{"source"})
	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "art-1",
				DesiredProfiles: []registry_dto.NamedProfile{
					{
						Name: "thumb",
						Profile: registry_dto.DesiredProfile{
							CapabilityName: "resize",
							Priority:       registry_dto.PriorityNeed,
							DependsOn:      sourceDeps,
						},
					},
					{
						Name: "webp",
						Profile: registry_dto.DesiredProfile{
							CapabilityName: "convert",
							Priority:       registry_dto.PriorityWant,
							DependsOn:      sourceDeps,
						},
					},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
	}

	dispatcher := &mockTaskDispatcher{}
	bridge := NewArtefactWorkflowBridge(reg, nil, dispatcher, nil)
	ctx := t.Context()

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("artefact.created"),
		Payload: map[string]any{"artefactID": "art-1"},
	}

	bridge.handleEvent(ctx, event)
	require.Len(t, dispatcher.dispatched, 2)

	thumbTask := dispatcher.dispatched[0]
	assert.Equal(t, "art-1:thumb", thumbTask.DeduplicationKey)
	assert.Equal(t, orchestrator_domain.PriorityHigh, thumbTask.Config.Priority)

	webpTask := dispatcher.dispatched[1]
	assert.Equal(t, "art-1:webp", webpTask.DeduplicationKey)
	assert.Equal(t, orchestrator_domain.PriorityLow, webpTask.Config.Priority)
}

func TestHandleEventWithAck_ValidArtefactID(t *testing.T) {
	t.Parallel()

	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:              "art-valid",
				DesiredProfiles: []registry_dto.NamedProfile{},
				ActualVariants:  []registry_dto.Variant{},
			}, nil
		},
	}

	bridge := NewArtefactWorkflowBridge(reg, nil, nil, nil)
	ctx := t.Context()

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("artefact.created"),
		Payload: map[string]any{"artefactID": "art-valid"},
	}

	err := bridge.handleEventWithAck(ctx, event)
	assert.NoError(t, err)
}

func TestListen_HandlerBased(t *testing.T) {
	t.Parallel()

	bridge := NewArtefactWorkflowBridge(nil, nil, nil, nil)
	eventBus := newMockEventBus()

	ctx, cancel := context.WithCancelCause(t.Context())

	done := make(chan struct{})
	go func() {
		bridge.Listen(ctx, eventBus)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel(fmt.Errorf("test: simulating cancelled context"))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Listen did not return after context cancellation")
	}

	for _, topic := range registry_domain.ArtefactTopics {
		assert.GreaterOrEqual(t, eventBus.getHandlerCount(topic), 1)
	}
}

func TestListen_ChannelBased(t *testing.T) {
	t.Parallel()

	bridge := NewArtefactWorkflowBridge(nil, nil, nil, nil)
	simpleBus := &simpleEventBus{}

	ctx, cancel := context.WithCancelCause(t.Context())

	done := make(chan struct{})
	go func() {
		bridge.Listen(ctx, simpleBus)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel(fmt.Errorf("test: simulating cancelled context"))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Listen did not return after context cancellation")
	}
}

func TestStartListening_HandlerBased(t *testing.T) {
	t.Parallel()

	bridge := NewArtefactWorkflowBridge(nil, nil, nil, nil)
	eventBus := newMockEventBus()

	ctx, cancel := context.WithCancelCause(t.Context())

	wait, err := bridge.StartListening(ctx, eventBus)
	require.NoError(t, err)
	require.NotNil(t, wait)

	done := make(chan struct{})
	go func() {
		wait()
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel(fmt.Errorf("test: simulating cancelled context"))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wait did not return after context cancellation")
	}
}

func TestStartListening_ChannelBased(t *testing.T) {
	t.Parallel()

	bridge := NewArtefactWorkflowBridge(nil, nil, nil, nil)
	channelBus := &channelEventBus{}

	ctx, cancel := context.WithCancelCause(t.Context())

	wait, err := bridge.StartListening(ctx, channelBus)
	require.NoError(t, err)
	require.NotNil(t, wait)

	done := make(chan struct{})
	go func() {
		wait()
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel(fmt.Errorf("test: simulating cancelled context"))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wait did not return after context cancellation")
	}
}

type channelEventBus struct{}

func (c *channelEventBus) Publish(_ context.Context, _ string, _ orchestrator_domain.Event) error {
	return nil
}

func (c *channelEventBus) Subscribe(ctx context.Context, _ string) (<-chan orchestrator_domain.Event, error) {
	eventChannel := make(chan orchestrator_domain.Event, 10)
	go func() {
		<-ctx.Done()
		close(eventChannel)
	}()
	return eventChannel, nil
}

func (c *channelEventBus) Close(_ context.Context) error {
	return nil
}

func (c *channelEventBus) SubscribeWithHandler(_ context.Context, _ string, _ orchestrator_domain.EventHandler) error {
	return nil
}

func TestCreateMessage(t *testing.T) {
	t.Parallel()

	web := &watermillEventBus{}
	ctx := t.Context()
	_, span, _ := log.Span(ctx, "test")
	defer span.End()

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("test.event"),
		Payload: map[string]any{"key": "value"},
	}

	message, err := web.createMessage(ctx, span, event)
	require.NoError(t, err)
	require.NotNil(t, message)

	assert.NotEmpty(t, message.UUID)
	assert.NotEmpty(t, message.Payload)
	assert.Equal(t, "test.event", message.Metadata.Get("event_type"))
	assert.NotEmpty(t, message.Metadata.Get("published_at"))
}

func TestCreateMessage_MarshalError(t *testing.T) {
	t.Parallel()

	web := &watermillEventBus{}
	ctx := t.Context()
	_, span, _ := log.Span(ctx, "test")
	defer span.End()

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("test.event"),
		Payload: map[string]any{"badvalue": make(chan int)},
	}

	_, err := web.createMessage(ctx, span, event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshalling event")
}

func TestCompilerExecutor_Execute_Success(t *testing.T) {
	t.Parallel()

	blobStore := &registry_domain.MockBlobStore{
		PutFunc: drainReader,
	}
	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	var resultTags registry_dto.Tags
	resultTags.SetByName("storageBackendId", "default")
	resultTags.SetByName("mimeType", "image/webp")
	resultTags.SetByName("fileExtension", ".webp")

	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:         "art-1",
				SourcePath: "images/photo.jpg",
				DesiredProfiles: []registry_dto.NamedProfile{
					{
						Name: "thumb",
						Profile: registry_dto.DesiredProfile{
							CapabilityName: "resize",
							DependsOn:      deps,
							ResultingTags:  resultTags,
						},
					},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady, StorageKey: "images/photo.jpg"},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("source data"))), nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return blobStore, nil
		},
	}

	capService := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return bytes.NewReader([]byte("compressed output")), nil
		},
	}

	executor := NewCompilerExecutor(reg, capService)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-1",
		"sourceVariantID":    "source",
		"desiredProfileName": "thumb",
		"capabilityToRun":    "resize",
		"taskID":             "task-1",
		"capabilityParams":   map[string]string{"width": "100"},
	}

	result, err := executor.Execute(ctx, payload)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "SUCCESS", result["status"])
	assert.NotEmpty(t, result["variantId"])
	assert.NotEmpty(t, result["storageKey"])
}

func TestCompilerExecutor_Execute_InvalidPayload(t *testing.T) {
	t.Parallel()

	executor := NewCompilerExecutor(nil, nil)
	ctx := t.Context()

	payload := map[string]any{
		"missing": "required fields",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid payload")
}

func TestCompilerExecutor_Execute_ArtefactNotFound(t *testing.T) {
	t.Parallel()

	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("not found")
		},
	}

	executor := NewCompilerExecutor(reg, nil)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-missing",
		"sourceVariantID":    "source",
		"desiredProfileName": "thumb",
		"capabilityToRun":    "resize",
		"taskID":             "task-1",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetching task metadata")
}

func TestCompilerExecutor_Execute_SourceVariantNotFound(t *testing.T) {
	t.Parallel()

	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "art-1",
				DesiredProfiles: []registry_dto.NamedProfile{
					{Name: "thumb", Profile: registry_dto.DesiredProfile{}},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "other"},
				},
			}, nil
		},
	}

	executor := NewCompilerExecutor(reg, nil)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-1",
		"sourceVariantID":    "source",
		"desiredProfileName": "thumb",
		"capabilityToRun":    "resize",
		"taskID":             "task-1",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetching task metadata")
}

func TestCompilerExecutor_Execute_CapabilityFails(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "art-1",
				DesiredProfiles: []registry_dto.NamedProfile{
					{Name: "thumb", Profile: registry_dto.DesiredProfile{
						CapabilityName: "resize",
						DependsOn:      deps,
					}},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("source data"))), nil
		},
	}

	capService := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return nil, errors.New("capability failed")
		},
	}

	executor := NewCompilerExecutor(reg, capService)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-1",
		"sourceVariantID":    "source",
		"desiredProfileName": "thumb",
		"capabilityToRun":    "resize",
		"taskID":             "task-1",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "capability")
}

func TestCompilerExecutor_Execute_GetVariantDataFails(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "art-1",
				DesiredProfiles: []registry_dto.NamedProfile{
					{Name: "thumb", Profile: registry_dto.DesiredProfile{
						CapabilityName: "resize",
						DependsOn:      deps,
					}},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return nil, errors.New("storage unavailable")
		},
	}

	executor := NewCompilerExecutor(reg, nil)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-1",
		"sourceVariantID":    "source",
		"desiredProfileName": "thumb",
		"capabilityToRun":    "resize",
		"taskID":             "task-1",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get source data")
}

func TestCompilerExecutor_Execute_GetBlobStoreFails(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	var resultTags registry_dto.Tags
	resultTags.SetByName("storageBackendId", "default")

	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:         "art-1",
				SourcePath: "images/photo.jpg",
				DesiredProfiles: []registry_dto.NamedProfile{
					{Name: "thumb", Profile: registry_dto.DesiredProfile{
						CapabilityName: "resize",
						DependsOn:      deps,
						ResultingTags:  resultTags,
					}},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("source data"))), nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return nil, errors.New("blob store unavailable")
		},
	}

	capService := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return bytes.NewReader([]byte("output")), nil
		},
	}

	executor := NewCompilerExecutor(reg, capService)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-1",
		"sourceVariantID":    "source",
		"desiredProfileName": "thumb",
		"capabilityToRun":    "resize",
		"taskID":             "task-1",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "storing and creating variant")
}

func TestCompilerExecutor_Execute_PutBlobFails(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	var resultTags registry_dto.Tags
	resultTags.SetByName("storageBackendId", "default")
	resultTags.SetByName("fileExtension", ".webp")

	blobStore := &registry_domain.MockBlobStore{
		PutFunc: func(_ context.Context, _ string, _ io.Reader) error {
			return errors.New("storage write failed")
		},
	}

	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:         "art-1",
				SourcePath: "images/photo.jpg",
				DesiredProfiles: []registry_dto.NamedProfile{
					{Name: "thumb", Profile: registry_dto.DesiredProfile{
						CapabilityName: "resize",
						DependsOn:      deps,
						ResultingTags:  resultTags,
					}},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("source data"))), nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return blobStore, nil
		},
	}

	capService := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return bytes.NewReader([]byte("output")), nil
		},
	}

	executor := NewCompilerExecutor(reg, capService)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-1",
		"sourceVariantID":    "source",
		"desiredProfileName": "thumb",
		"capabilityToRun":    "resize",
		"taskID":             "task-1",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "storing and creating variant")
}

func TestCompilerExecutor_Execute_RenameBlobFails(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	var resultTags registry_dto.Tags
	resultTags.SetByName("storageBackendId", "default")
	resultTags.SetByName("fileExtension", ".webp")

	blobStore := &registry_domain.MockBlobStore{
		PutFunc: drainReader,
		RenameFunc: func(_ context.Context, _ string, _ string) error {
			return errors.New("rename failed")
		},
	}

	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:         "art-1",
				SourcePath: "images/photo.jpg",
				DesiredProfiles: []registry_dto.NamedProfile{
					{Name: "thumb", Profile: registry_dto.DesiredProfile{
						CapabilityName: "resize",
						DependsOn:      deps,
						ResultingTags:  resultTags,
					}},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("source data"))), nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return blobStore, nil
		},
	}

	capService := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return bytes.NewReader([]byte("output")), nil
		},
	}

	executor := NewCompilerExecutor(reg, capService)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-1",
		"sourceVariantID":    "source",
		"desiredProfileName": "thumb",
		"capabilityToRun":    "resize",
		"taskID":             "task-1",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "storing and creating variant")
}

func TestCompilerExecutor_Execute_AddVariantFails(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	var resultTags registry_dto.Tags
	resultTags.SetByName("storageBackendId", "default")
	resultTags.SetByName("mimeType", "image/webp")
	resultTags.SetByName("fileExtension", ".webp")

	blobStore := &registry_domain.MockBlobStore{
		PutFunc: drainReader,
	}

	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:         "art-1",
				SourcePath: "images/photo.jpg",
				DesiredProfiles: []registry_dto.NamedProfile{
					{Name: "thumb", Profile: registry_dto.DesiredProfile{
						CapabilityName: "resize",
						DependsOn:      deps,
						ResultingTags:  resultTags,
					}},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("source data"))), nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return blobStore, nil
		},
		AddVariantFunc: func(_ context.Context, _ string, _ *registry_dto.Variant) (*registry_dto.ArtefactMeta, error) {
			return nil, errors.New("add variant failed")
		},
	}

	capService := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return bytes.NewReader([]byte("compressed output")), nil
		},
	}

	executor := NewCompilerExecutor(reg, capService)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-1",
		"sourceVariantID":    "source",
		"desiredProfileName": "thumb",
		"capabilityToRun":    "resize",
		"taskID":             "task-1",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add variant")
}

func TestSetupHandlerSubscriptions_SubscribeFails(t *testing.T) {
	t.Parallel()

	bridge := NewArtefactWorkflowBridge(nil, nil, nil, nil)
	ctx := t.Context()

	failBus := &failingHandlerEventBus{}

	wait, err := bridge.setupHandlerSubscriptions(ctx, failBus)
	require.Error(t, err)
	assert.Nil(t, wait)
	assert.Contains(t, err.Error(), "subscribing to topic")
}

type failingHandlerEventBus struct{}

func (f *failingHandlerEventBus) Publish(_ context.Context, _ string, _ orchestrator_domain.Event) error {
	return nil
}

func (f *failingHandlerEventBus) Subscribe(_ context.Context, _ string) (<-chan orchestrator_domain.Event, error) {
	return nil, nil
}

func (f *failingHandlerEventBus) Close(_ context.Context) error {
	return nil
}

func (f *failingHandlerEventBus) SubscribeWithHandler(_ context.Context, _ string, _ orchestrator_domain.EventHandler) error {
	return assert.AnError
}

func TestCompilerExecutor_Execute_DesiredProfileNotFound(t *testing.T) {
	t.Parallel()

	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "art-1",
				DesiredProfiles: []registry_dto.NamedProfile{
					{Name: "other", Profile: registry_dto.DesiredProfile{}},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
	}

	executor := NewCompilerExecutor(reg, nil)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-1",
		"sourceVariantID":    "source",
		"desiredProfileName": "thumb",
		"capabilityToRun":    "resize",
		"taskID":             "task-1",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetching task metadata")
}

func TestCompilerExecutor_Execute_CapabilityFatalError(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "art-fatal",
				DesiredProfiles: []registry_dto.NamedProfile{
					{Name: "compiled", Profile: registry_dto.DesiredProfile{
						CapabilityName: "compile-component",
						DependsOn:      deps,
					}},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("source data"))), nil
		},
	}

	capService := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return nil, capabilities_domain.NewFatalError(errors.New("parse error: invalid syntax"))
		},
	}

	executor := NewCompilerExecutor(reg, capService)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-fatal",
		"sourceVariantID":    "source",
		"desiredProfileName": "compiled",
		"capabilityToRun":    "compile-component",
		"taskID":             "task-fatal-cap",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse error: invalid syntax")

	assert.True(t, orchestrator_domain.IsFatalError(err),
		"executor should translate capability fatal to orchestrator fatal")

	assert.True(t, capabilities_domain.IsFatalError(err),
		"original capability fatal must be preserved in the error chain")
}

type fatalLazyReader struct {
	err error
}

func (r *fatalLazyReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

func TestCompilerExecutor_Execute_LazyFatalError(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	var resultTags registry_dto.Tags
	resultTags.SetByName("storageBackendId", "default")
	resultTags.SetByName("fileExtension", ".min.js")

	blobStore := &registry_domain.MockBlobStore{
		PutFunc: func(_ context.Context, _ string, r io.Reader) error {
			buffer := make([]byte, 1024)
			_, err := r.Read(buffer)
			if err != nil {
				return err
			}
			return nil
		},
	}

	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID:         "art-lazy-fatal",
				SourcePath: "components/broken.pkc",
				DesiredProfiles: []registry_dto.NamedProfile{
					{Name: "minified", Profile: registry_dto.DesiredProfile{
						CapabilityName: "minify-js",
						DependsOn:      deps,
						ResultingTags:  resultTags,
					}},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("const v = 1;"))), nil
		},
		GetBlobStoreFunc: func(_ string) (registry_domain.BlobStore, error) {
			return blobStore, nil
		},
	}

	capService := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return &fatalLazyReader{
				err: capabilities_domain.NewFatalError(
					errors.New("identifier v has already been declared")),
			}, nil
		},
	}

	executor := NewCompilerExecutor(reg, capService)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-lazy-fatal",
		"sourceVariantID":    "source",
		"desiredProfileName": "minified",
		"capabilityToRun":    "minify-js",
		"taskID":             "task-lazy-fatal",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "identifier v has already been declared")

	assert.True(t, orchestrator_domain.IsFatalError(err),
		"lazy fatal error should be translated to orchestrator fatal")

	assert.True(t, capabilities_domain.IsFatalError(err),
		"original capability fatal must be preserved in the error chain")
}

func TestCompilerExecutor_Execute_NonFatalCapabilityError(t *testing.T) {
	t.Parallel()

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	reg := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
			return &registry_dto.ArtefactMeta{
				ID: "art-nonfatal",
				DesiredProfiles: []registry_dto.NamedProfile{
					{Name: "compiled", Profile: registry_dto.DesiredProfile{
						CapabilityName: "compile-component",
						DependsOn:      deps,
					}},
				},
				ActualVariants: []registry_dto.Variant{
					{VariantID: "source", Status: registry_dto.VariantStatusReady},
				},
			}, nil
		},
		GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader([]byte("source data"))), nil
		},
	}

	capService := &capabilities_domain.MockCapabilityService{
		ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
			return nil, errors.New("transient resource exhaustion")
		},
	}

	executor := NewCompilerExecutor(reg, capService)
	ctx := t.Context()

	payload := map[string]any{
		"artefactID":         "art-nonfatal",
		"sourceVariantID":    "source",
		"desiredProfileName": "compiled",
		"capabilityToRun":    "compile-component",
		"taskID":             "task-nonfatal",
	}

	_, err := executor.Execute(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transient resource exhaustion")

	assert.False(t, orchestrator_domain.IsFatalError(err),
		"non-fatal capability error should not become orchestrator fatal")
}
