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

// Package piko provides the public API for the Piko web framework.
//
// This is the primary entry point for applications built with Piko. It
// exposes a unified facade over the framework's internal subsystems,
// including server lifecycle management, server actions, component
// rendering, collections, search, health monitoring, internationalisation,
// linguistics, and testing.
//
// # Getting started
//
// Create a server, optionally configure it, and call Run:
//
//	server := piko.New()
//	server.Configure(piko.PublicConfig{Port: 8080})
//	if err := server.Run(piko.RunModeDev); err != nil {
//	    log.Fatal(err)
//	}
//
// # Server actions
//
// Actions handle client-initiated mutations. Actions embed
// [ActionMetadata] and define a Call method with any signature:
//
//	type DeleteAction struct {
//	    piko.ActionMetadata
//	}
//
//	func (a DeleteAction) Call(id int64) (Response, error) {
//	    return Response{Success: true}, nil
//	}
//
// Structured error types ([ValidationError], [NotFoundError],
// [ForbiddenError], etc.) are automatically mapped to HTTP status
// codes by the action handler.
//
// # Collections and search
//
// Use [GetData], [GetSections], and [GetSectionsTree] to extract
// typed data and table-of-contents headings from collection content.
// [SearchCollection] provides fuzzy full-text search with field
// weighting, and [QuickSearch] offers a convenience wrapper with
// sensible defaults.
//
// # Lifecycle and health
//
// Register external components for managed startup/shutdown with
// [SSRServer.RegisterLifecycle]. Components that also implement
// [LifecycleHealthProbe] are automatically added to the health
// monitoring system. Custom health probes can also be registered
// via [WithCustomHealthProbe].
//
// # Testing
//
// The package provides a testing framework for components and
// actions. Use [NewComponentTester] to render and assert against
// a component's AST, and [NewActionTester] to invoke and verify
// server actions. [NewTestRequest] builds [RequestData] with a
// fluent API for injecting context, query parameters, and form
// data.
//
// # Thread safety
//
// [SSRServer] methods must be called from a single goroutine
// during setup. Once [SSRServer.Run] is called, the underlying
// HTTP server and all registered services are safe for concurrent
// use. Collection query functions ([GetData], [SearchCollection],
// [GetAllCollectionItems]) are safe to call concurrently.
package piko
