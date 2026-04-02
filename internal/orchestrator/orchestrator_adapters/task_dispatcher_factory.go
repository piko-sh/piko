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

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

// CreateTaskDispatcher creates a TaskDispatcher for distributed task processing.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes config (DispatcherConfig) which specifies the dispatcher settings
// including handler counts per priority level.
// Takes eventBus (EventBus) which provides pub/sub for task distribution
// via Watermill topics.
// Takes taskStore (TaskStore) which provides persistence and crash recovery.
//
// Returns TaskDispatcher which is ready to have executors registered and be
// started, or nil if eventBus is nil.
func CreateTaskDispatcher(
	ctx context.Context,
	config orchestrator_domain.DispatcherConfig,
	eventBus orchestrator_domain.EventBus,
	taskStore orchestrator_domain.TaskStore,
) orchestrator_domain.TaskDispatcher {
	ctx, fl := logger_domain.From(ctx, log)
	if eventBus == nil {
		fl.Error("EventBus is nil, cannot create TaskDispatcher")
		return nil
	}

	fl.Internal("Creating Watermill-based task dispatcher for distributed processing",
		logger_domain.Int("highHandlers", config.WatermillHighHandlers),
		logger_domain.Int("normalHandlers", config.WatermillNormalHandlers),
		logger_domain.Int("lowHandlers", config.WatermillLowHandlers))

	return newWatermillTaskDispatcher(config, eventBus, taskStore)
}

// eventBusTypeName returns a type name for logging purposes.
//
// Takes eventBus (EventBus) which is the event bus to identify.
//
// Returns string which is the type name, or "nil" if the event bus is nil.
func eventBusTypeName(eventBus orchestrator_domain.EventBus) string {
	if eventBus == nil {
		return "nil"
	}
	switch eventBus.(type) {
	case *watermillEventBus:
		return "watermillEventBus"
	default:
		return "unknown"
	}
}
