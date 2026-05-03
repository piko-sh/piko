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
	"time"

	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// logKeyArtefactID is the log field key for artefact identifiers.
	logKeyArtefactID = "artefactID"

	// logKeyEventType is the structured log key for the event type field.
	logKeyEventType = "eventType"

	// logKeyTopicCount is the log field key for the number of topics.
	logKeyTopicCount = "topicCount"

	// logKeyReference is the log field key for the related component name.
	logKeyReference = "reference"
)

// ArtefactWorkflowBridge is a driving adapter that listens for artefact
// events from the registry and dispatches compilation tasks to the
// orchestrator. It implements BridgeWithCounter and bridges the registry
// hexagon's domain events to the orchestrator's task system.
//
// Deduplication: Tasks are deduplicated at the database level using
// DeduplicationKey (artefactID:profileName). This limits to one active
// task exists per profile across all instances, supporting distributed
// deployments.
type ArtefactWorkflowBridge struct {
	// registryService provides access to the artefact registry for lookups.
	registryService registry_domain.RegistryService

	// orchestratorService manages the running of multi-step workflows.
	orchestratorService orchestrator_domain.OrchestratorService

	// taskDispatcher sends tasks to priority queues for processing at the same
	// time.
	taskDispatcher orchestrator_domain.TaskDispatcher

	// eventBus sends events to handlers that have registered to receive them.
	eventBus orchestrator_domain.EventBus

	// eventGroup prevents duplicate processing of events for the same artefact.
	eventGroup singleflight.Group

	// artefactEventsProcessed counts the number of artefact events fully handled.
	// The daemon uses this for flush detection: it waits until this count
	// matches the registry's published count before checking if the dispatcher
	// is idle.
	artefactEventsProcessed atomic.Int64
}

// NewArtefactWorkflowBridge creates a new ArtefactWorkflowBridge with the
// given dependencies.
//
// Takes registry (RegistryService) which provides artefact registry access.
// Takes orchestrator (OrchestratorService) which manages workflow execution.
// Takes taskDispatcher (TaskDispatcher) which dispatches tasks to workers.
// Takes eventBus (EventBus) which handles event distribution.
//
// Returns *ArtefactWorkflowBridge which is ready for use.
func NewArtefactWorkflowBridge(
	registry registry_domain.RegistryService,
	orchestrator orchestrator_domain.OrchestratorService,
	taskDispatcher orchestrator_domain.TaskDispatcher,
	eventBus orchestrator_domain.EventBus,
) *ArtefactWorkflowBridge {
	return &ArtefactWorkflowBridge{
		registryService:         registry,
		orchestratorService:     orchestrator,
		taskDispatcher:          taskDispatcher,
		eventBus:                eventBus,
		eventGroup:              singleflight.Group{},
		artefactEventsProcessed: atomic.Int64{},
	}
}

// Listen starts listening for artefact events from the provided EventBus,
// automatically detecting handler-based subscription support for efficient Ack/Nack
// semantics.
//
// Behaviour: blocks until the context is cancelled. For proper initialisation
// ordering, use StartListening() instead which sets up subscriptions synchronously
// and returns immediately.
//
// When using handler-based subscription:
//   - Messages are processed by Watermill's router
//   - Returning nil from the handler Acks the message
//   - Returning error from the handler Nacks the message
//   - Singleflight provides idempotent processing per artefact
//
// Takes bus (orchestrator_domain.EventBus) which provides the
// event source to subscribe to.
func (b *ArtefactWorkflowBridge) Listen(ctx context.Context, bus orchestrator_domain.EventBus) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "ArtefactWorkflowBridge.Listen",
		logger_domain.String(logKeyReference, "eventBus"),
	)
	defer span.End()

	b.listenWithHandler(ctx, bus)
}

// StartListening sets up subscriptions synchronously and returns
// immediately, providing a wait function that blocks until the context
// is cancelled so that subscriptions are established before any events
// are published.
//
// Usage:
// wait, err := bridge.StartListening(ctx, bus)
//
//	if err != nil {
//	    return err
//	}
//
// go wait() // Block in goroutine until shutdown
// This is the preferred method over Listen() for proper initialisation
// ordering.
//
// Takes bus (orchestrator_domain.EventBus) which provides the event source
// to subscribe to.
//
// Returns wait (func()) which blocks until the context is
// cancelled.
// Returns err (error) when subscription setup fails.
func (b *ArtefactWorkflowBridge) StartListening(ctx context.Context, bus orchestrator_domain.EventBus) (wait func(), err error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "ArtefactWorkflowBridge.StartListening",
		logger_domain.String(logKeyReference, "eventBus"),
	)
	defer span.End()

	return b.setupHandlerSubscriptions(ctx, bus)
}

// ArtefactEventsProcessed returns the number of artefact events fully
// processed. Used for pipeline flush detection - the daemon waits until this
// equals the registry's published count before checking if the dispatcher is
// idle.
//
// Returns int64 which is the count of fully processed artefact events.
func (b *ArtefactWorkflowBridge) ArtefactEventsProcessed() int64 {
	return b.artefactEventsProcessed.Load()
}

// setupHandlerSubscriptions establishes handler-based subscriptions
// synchronously and returns a wait function that blocks until context
// cancellation.
//
// Takes ctx (context.Context) which controls the subscription lifetime.
// Takes bus (orchestrator_domain.EventBus) which provides
// handler-based event subscription.
//
// Returns func() which blocks until the context is cancelled.
// Returns error when subscribing to any artefact topic fails.
func (b *ArtefactWorkflowBridge) setupHandlerSubscriptions(ctx context.Context, bus orchestrator_domain.EventBus) (func(), error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "ArtefactWorkflowBridge.setupHandlerSubscriptions")
	defer span.End()

	l.Trace("Subscribing to artefact events with handler",
		logger_domain.Int(logKeyTopicCount, len(registry_domain.ArtefactTopics)))

	for _, topic := range registry_domain.ArtefactTopics {
		topicCopy := topic
		err := bus.SubscribeWithHandler(ctx, topicCopy, func(eventCtx context.Context, event orchestrator_domain.Event) error {
			l.Trace("Received artefact event",
				logger_domain.String(logKeyTopic, topicCopy),
				logger_domain.String(logKeyEventType, string(event.Type)),
				logger_domain.Int("payloadSize", len(event.Payload)))

			BridgeEventsProcessedCount.Add(eventCtx, 1)

			return b.handleEventWithAck(eventCtx, event)
		})

		if err != nil {
			l.ReportError(span, err, "Failed to subscribe with handler",
				logger_domain.String(logKeyTopic, topicCopy))
			BridgeEventHandlingErrorCount.Add(ctx, 1)
			return nil, fmt.Errorf("subscribing to topic %q: %w", topicCopy, err)
		}

		l.Trace("Subscribed to topic", logger_domain.String(logKeyTopic, topicCopy))
	}

	l.Internal("Handler subscriptions established",
		logger_domain.Int(logKeyTopicCount, len(registry_domain.ArtefactTopics)),
		logger_domain.Strings("topics", registry_domain.ArtefactTopics))

	return func() {
		<-ctx.Done()
		l.Trace("Context cancelled, handler subscriptions will be cleaned up by EventBus")
	}, nil
}

// listenWithHandler subscribes to artefact events using handler-based
// semantics for proper Ack/Nack support.
//
// Preferred when using a Watermill-backed event bus. Unlike channel-based
// subscription, Watermill's GoChannel does not support wildcard patterns. Subscribes
// to each topic in registry_domain.ArtefactTopics explicitly.
//
// Takes bus (orchestrator_domain.EventBus) which provides the event
// bus with handler-based subscription support.
func (b *ArtefactWorkflowBridge) listenWithHandler(ctx context.Context, bus orchestrator_domain.EventBus) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "ArtefactWorkflowBridge.listenWithHandler")
	defer span.End()

	l.Trace("Subscribing to artefact events with handler",
		logger_domain.Int(logKeyTopicCount, len(registry_domain.ArtefactTopics)))

	for _, topic := range registry_domain.ArtefactTopics {
		topicCopy := topic
		err := bus.SubscribeWithHandler(ctx, topicCopy, func(eventCtx context.Context, event orchestrator_domain.Event) error {
			l.Trace("Received event via handler",
				logger_domain.String(logKeyTopic, topicCopy),
				logger_domain.String(logKeyEventType, string(event.Type)),
				logger_domain.Int("payloadSize", len(event.Payload)))

			BridgeEventsProcessedCount.Add(eventCtx, 1)

			return b.handleEventWithAck(eventCtx, event)
		})

		if err != nil {
			l.ReportError(span, err, "Failed to subscribe with handler",
				logger_domain.String(logKeyTopic, topicCopy))
			BridgeEventHandlingErrorCount.Add(ctx, 1)
			return
		}

		l.Trace("Subscribed to topic", logger_domain.String(logKeyTopic, topicCopy))
	}

	l.Internal("Handler subscriptions established",
		logger_domain.Int(logKeyTopicCount, len(registry_domain.ArtefactTopics)),
		logger_domain.Strings("topics", registry_domain.ArtefactTopics))

	<-ctx.Done()
	l.Trace("Context cancelled, handler subscriptions will be cleaned up by EventBus")
}

// handleEventWithAck wraps handleEvent to return Ack or Nack signals.
//
// Takes event (orchestrator_domain.Event) which contains the event to process.
//
// Returns error when the message should be Nacked for redelivery; returns nil
// to Ack when the message was processed or is not valid.
func (b *ArtefactWorkflowBridge) handleEventWithAck(ctx context.Context, event orchestrator_domain.Event) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "ArtefactWorkflowBridge.handleEventWithAck",
		logger_domain.String(logKeyEventType, string(event.Type)),
	)
	defer span.End()

	artefactID, ok := event.Payload[logKeyArtefactID].(string)
	if !ok || artefactID == "" {
		l.Warn("Received event with missing or invalid artefactID, Acking to prevent redelivery")
		BridgeEventHandlingErrorCount.Add(ctx, 1)
		return nil
	}

	b.handleEvent(ctx, event)

	return nil
}

// handleEvent processes an artefact workflow event.
//
// Takes event (orchestrator_domain.Event) which contains the event type and
// payload with artefact details.
func (b *ArtefactWorkflowBridge) handleEvent(ctx context.Context, event orchestrator_domain.Event) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "ArtefactWorkflowBridge.handleEvent",
		logger_domain.String(logKeyReference, "artefactEvent"),
		logger_domain.String(logKeyEventType, string(event.Type)),
		logger_domain.Int("payloadSize", len(event.Payload)),
	)
	defer span.End()

	l.Trace("Processing artefact event",
		logger_domain.String(logKeyEventType, string(event.Type)),
		logger_domain.Field("payload", event.Payload))

	startTime := time.Now()

	artefactID, ok := event.Payload[logKeyArtefactID].(string)
	if !ok || artefactID == "" {
		l.Warn("Received event with missing or invalid artefactID",
			logger_domain.Field("payload", event.Payload))
		l.ReportError(span, errors.New("missing or invalid artefactID"), "Missing or invalid artefactID")
		BridgeEventHandlingErrorCount.Add(ctx, 1)
		return
	}

	l = l.With(logger_domain.String(logKeyArtefactID, artefactID))
	sfKey := string(event.Type) + ":" + artefactID

	_, err, _ := b.eventGroup.Do(sfKey, func() (any, error) {
		return b.processArtefactEvent(ctx, span, event, artefactID)
	})

	b.finaliseEventHandling(ctx, span, err, artefactID, startTime)
}

// processArtefactEvent handles the main logic for processing an artefact event
// within singleflight.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes span (trace.Span) which records tracing data for this operation.
// Takes event (orchestrator_domain.Event) which contains the event to process.
// Takes artefactID (string) which identifies the artefact to process.
//
// Returns any which is always nil.
// Returns error when fetching the artefact fails.
func (b *ArtefactWorkflowBridge) processArtefactEvent(
	ctx context.Context,
	span trace.Span,
	event orchestrator_domain.Event,
	artefactID string,
) (any, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Singleflight: Processing unique artefact event",
		logger_domain.String("key", string(event.Type)+":"+artefactID))

	if event.Type == registry_domain.EventArtefactDeleted {
		l.Trace("Artefact deleted, no further processing needed")
		span.SetStatus(codes.Ok, "Artefact deleted, no further processing needed")
		return nil, nil
	}

	artefact, err := b.fetchArtefact(ctx, artefactID)
	if err != nil {
		l.ReportError(span, err, "Failed to get artefact after receiving event")
		BridgeArtefactFetchErrorCount.Add(ctx, 1)
		BridgeEventHandlingErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("fetching artefact %q: %w", artefactID, err)
	}

	variantStatus := buildVariantStatusMap(artefact.ActualVariants)

	l.Trace("Evaluating desired profiles",
		logger_domain.Int("profileCount", len(artefact.DesiredProfiles)),
		logger_domain.Int("existingVariantCount", len(artefact.ActualVariants)),
	)
	span.SetAttributes(
		attribute.Int("variantCount", len(artefact.ActualVariants)),
		attribute.Int("profileCount", len(artefact.DesiredProfiles)),
	)

	tasksDispatched := b.evaluateAndDispatchProfiles(ctx, artefact, variantStatus)
	span.SetAttributes(attribute.Int("tasksDispatched", tasksDispatched))

	return nil, nil
}

// fetchArtefact retrieves artefact details from the registry and records
// fetch timing metrics.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes artefactID (string) which identifies the artefact to fetch.
//
// Returns *registry_dto.ArtefactMeta which contains the artefact details.
// Returns error when the registry service fails to retrieve the artefact.
func (b *ArtefactWorkflowBridge) fetchArtefact(
	ctx context.Context,
	artefactID string,
) (*registry_dto.ArtefactMeta, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Fetching artefact details from registry")

	var artefact *registry_dto.ArtefactMeta
	var fetchErr error

	err := l.RunInSpan(ctx, "FetchArtefact", func(ctx context.Context, _ logger_domain.Logger) error {
		fetchStartTime := time.Now()
		artefact, fetchErr = b.registryService.GetArtefact(ctx, artefactID)
		BridgeArtefactFetchDuration.Record(ctx, float64(time.Since(fetchStartTime).Milliseconds()))
		if fetchErr != nil {
			return fmt.Errorf("fetching artefact %q from registry: %w", artefactID, fetchErr)
		}
		return nil
	})

	return artefact, err
}

// evaluateAndDispatchProfiles iterates over all desired profiles and
// dispatches tasks for eligible ones.
//
// Takes artefact (*registry_dto.ArtefactMeta) which provides the artefact
// metadata containing the desired profiles to evaluate.
// Takes variantStatus (map[string]registry_dto.VariantStatus) which maps
// variant names to their current status.
//
// Returns int which is the number of tasks successfully dispatched.
func (b *ArtefactWorkflowBridge) evaluateAndDispatchProfiles(
	ctx context.Context,
	artefact *registry_dto.ArtefactMeta,
	variantStatus map[string]registry_dto.VariantStatus,
) int {
	tasksDispatched := 0

	for i := range artefact.DesiredProfiles {
		np := &artefact.DesiredProfiles[i]
		if b.evaluateAndDispatchProfile(ctx, artefact, np.Name, &np.Profile, variantStatus) {
			tasksDispatched++
		}
	}

	return tasksDispatched
}

// evaluateAndDispatchProfile evaluates a single profile and dispatches a task
// if dependencies are met.
//
// Takes artefact (*registry_dto.ArtefactMeta) which provides the artefact
// metadata to evaluate.
// Takes profileName (string) which identifies the profile to evaluate.
// Takes profile (registry_dto.DesiredProfile) which specifies the desired
// profile configuration.
// Takes variantStatus (map[string]registry_dto.VariantStatus) which provides
// the current status of all variants.
//
// Returns bool which is true if a task was successfully dispatched.
func (b *ArtefactWorkflowBridge) evaluateAndDispatchProfile(
	ctx context.Context,
	artefact *registry_dto.ArtefactMeta,
	profileName string,
	profile *registry_dto.DesiredProfile,
	variantStatus map[string]registry_dto.VariantStatus,
) bool {
	ctx, l := logger_domain.From(ctx, log)
	profileCtx, profileSpan, profileLogger := l.Span(ctx, "EvaluateProfile",
		logger_domain.String("profile", profileName),
		logger_domain.String("capability", profile.CapabilityName),
	)
	defer profileSpan.End()

	dedupKey := artefact.ID + ":" + profileName

	if isProfileAlreadyReady(variantStatus, profileName) {
		profileLogger.Trace("Profile variant already READY, skipping",
			logger_domain.String("payloadKeyDeduplicationKey", dedupKey))
		return false
	}

	missingDeps := findMissingDependencies(&profile.DependsOn, variantStatus)
	if len(missingDeps) > 0 {
		profileLogger.Trace("Dependencies not met, skipping",
			logger_domain.String("payloadKeyDeduplicationKey", dedupKey),
			logger_domain.Field("missing", missingDeps))
		profileSpan.SetAttributes(attribute.StringSlice("missingDependencies", missingDeps))
		return false
	}

	return b.dispatchProfileTask(profileCtx, profileSpan, artefact, profileName, profile, dedupKey)
}

// dispatchProfileTask creates and sends a compilation task for the given
// profile.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes span (trace.Span) which records tracing data.
// Takes artefact (*registry_dto.ArtefactMeta) which is the artefact to compile.
// Takes profileName (string) which identifies the target profile.
// Takes profile (registry_dto.DesiredProfile) which defines the compilation
// settings.
// Takes dedupKey (string) which prevents duplicate tasks.
//
// Returns bool which is true when the task was sent, or false when the task
// already exists or sending failed.
func (b *ArtefactWorkflowBridge) dispatchProfileTask(
	ctx context.Context,
	span trace.Span,
	artefact *registry_dto.ArtefactMeta,
	profileName string,
	profile *registry_dto.DesiredProfile,
	dedupKey string,
) bool {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("All dependencies met, dispatching task",
		logger_domain.String("payloadKeyDeduplicationKey", dedupKey),
		logger_domain.String("artefactID", artefact.ID),
		logger_domain.String("profileName", profileName))

	task := orchestrator_domain.NewTask("artefact.compiler", map[string]any{
		"artefactID":         artefact.ID,
		"sourceVariantID":    profile.DependsOn.First(),
		"desiredProfileName": profileName,
		"capabilityToRun":    profile.CapabilityName,
		"capabilityParams":   profile.Params.ToMap(),
	})
	task.WorkflowID = artefact.ID
	task.Config.Priority = b.mapPriority(profile.Priority)
	task.DeduplicationKey = dedupKey
	task.Payload["taskID"] = task.ID

	if err := b.taskDispatcher.Dispatch(ctx, task); err != nil {
		if errors.Is(err, orchestrator_domain.ErrDuplicateTask) {
			l.Trace("Task already exists for profile, skipping duplicate",
				logger_domain.String(payloadKeyDeduplicationKey, dedupKey))
			orchestrator_domain.TaskDeduplicationBlockedCount.Add(ctx, 1)
			return false
		}
		l.ReportError(span, err, "Failed to dispatch task to priority topic")
		BridgeTaskDispatchErrorCount.Add(ctx, 1)
		return false
	}

	l.Trace("Task dispatched successfully",
		logger_domain.String("taskID", task.ID),
		logger_domain.String(payloadKeyDeduplicationKey, dedupKey))
	BridgeTasksDispatchedCount.Add(ctx, 1)
	return true
}

// finaliseEventHandling records metrics and updates span status after event
// processing completes.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes span (trace.Span) which receives status updates and timing data.
// Takes err (error) which shows whether event handling succeeded.
// Takes artefactID (string) which identifies the artefact for logging.
// Takes startTime (time.Time) which marks when processing began for metrics.
func (b *ArtefactWorkflowBridge) finaliseEventHandling(
	ctx context.Context,
	span trace.Span,
	err error,
	artefactID string,
	startTime time.Time,
) {
	ctx, l := logger_domain.From(ctx, log)
	if err != nil {
		l.Error("Event handling failed inside singleflight",
			logger_domain.String(logKeyArtefactID, artefactID),
			logger_domain.Error(err))
		span.SetStatus(codes.Error, "Event handling failed")
	} else {
		span.SetStatus(codes.Ok, "Event handled successfully")
	}

	duration := time.Since(startTime)
	BridgeEventHandlingDuration.Record(ctx, float64(duration.Milliseconds()))
	span.SetAttributes(attribute.Int64("totalDurationMs", duration.Milliseconds()))

	b.artefactEventsProcessed.Add(1)
}

// mapPriority converts a registry profile priority to a task priority.
//
// Takes priority (registry_dto.ProfilePriority) which is the source priority.
//
// Returns orchestrator_domain.TaskPriority which is the mapped task priority.
func (*ArtefactWorkflowBridge) mapPriority(priority registry_dto.ProfilePriority) orchestrator_domain.TaskPriority {
	ctx := context.Background()

	ctx, ml := logger_domain.From(ctx, log)
	ctx, span, l := ml.Span(ctx, "ArtefactWorkflowBridge.mapPriority",
		logger_domain.String(logKeyReference, "priorityMapping"),
		logger_domain.String("registryPriority", string(priority)),
	)
	defer span.End()

	var taskPriority orchestrator_domain.TaskPriority

	switch priority {
	case registry_dto.PriorityNeed:
		taskPriority = orchestrator_domain.PriorityHigh
		l.Trace("Mapped registry priority NEED to task priority HIGH")
		span.SetAttributes(attribute.String("taskPriority", "HIGH"))
	case registry_dto.PriorityWant:
		taskPriority = orchestrator_domain.PriorityLow
		l.Trace("Mapped registry priority WANT to task priority LOW")
		span.SetAttributes(attribute.String("taskPriority", "LOW"))
	default:
		taskPriority = orchestrator_domain.PriorityNormal
		l.Trace("Mapped registry priority to default task priority NORMAL")
		span.SetAttributes(attribute.String("taskPriority", "NORMAL"))
	}

	span.SetStatus(codes.Ok, "Priority mapped successfully")
	return taskPriority
}

// buildVariantStatusMap creates a lookup map from variant IDs to their status.
//
// Takes variants ([]registry_dto.Variant) which provides the variants to index.
//
// Returns map[string]registry_dto.VariantStatus which maps each variant ID to
// its status value.
func buildVariantStatusMap(variants []registry_dto.Variant) map[string]registry_dto.VariantStatus {
	variantStatus := make(map[string]registry_dto.VariantStatus, len(variants))
	for i := range variants {
		variantStatus[variants[i].VariantID] = variants[i].Status
	}
	return variantStatus
}

// isProfileAlreadyReady checks if a profile variant exists and is ready.
//
// Takes variantStatus (map[string]registry_dto.VariantStatus) which maps
// profile
// names to their current status.
// Takes profileName (string) which identifies the profile to check.
//
// Returns bool which is true when the profile exists and has ready status.
// Variants are only created with ready status after task completion.
// Database-level deduplication using DeduplicationKey handles the race window
// between task dispatch and variant creation.
func isProfileAlreadyReady(variantStatus map[string]registry_dto.VariantStatus, profileName string) bool {
	currentStatus, exists := variantStatus[profileName]
	return exists && currentStatus == registry_dto.VariantStatusReady
}

// findMissingDependencies returns a list of dependency IDs that are not ready.
//
// Takes dependsOn (*registry_dto.Dependencies) which contains the dependencies
// to check.
// Takes variantStatus (map[string]registry_dto.VariantStatus) which provides
// the current status of each variant.
//
// Returns []string which contains the IDs of dependencies that do not exist
// or are not in a ready state.
func findMissingDependencies(dependsOn *registry_dto.Dependencies, variantStatus map[string]registry_dto.VariantStatus) []string {
	var missingDeps []string
	for depID := range dependsOn.All() {
		depStatus, depExists := variantStatus[depID]
		if !depExists || depStatus != registry_dto.VariantStatusReady {
			missingDeps = append(missingDeps, depID)
		}
	}
	return missingDeps
}
