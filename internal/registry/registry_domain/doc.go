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

// Package registry_domain defines the core business logic for the
// artefact registry.
//
// It contains the service layer and port interfaces for managing
// artefacts, their variants, and blob storage. The service handles
// creation, updates, deletion, and variant management with
// content-addressable blob deduplication and reference counting.
//
// Lifecycle events are published via the event bus when artefacts are
// created, updated, or deleted.
//
// The service implementation is safe for concurrent use.
package registry_domain
