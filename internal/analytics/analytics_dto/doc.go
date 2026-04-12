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

// Package analytics_dto defines data transfer objects for the backend
// analytics subsystem.
//
// These types carry analytics event data across hexagonal boundaries,
// covering event classification, request metadata, and custom
// properties for third-party analytics backends.
//
// # Event types
//
// Three event types are defined: [EventPageView] for automatic page
// request tracking, [EventAction] for server action execution, and
// [EventCustom] for user-defined business events fired manually from
// action handlers.
//
// # Memory management
//
// [Event] instances are pooled via [AcquireEvent] and [ReleaseEvent]
// to avoid allocation on the hot request path. Collectors must not
// retain a pointer to the Event after Collect returns; they should
// copy any data they need.
package analytics_dto
