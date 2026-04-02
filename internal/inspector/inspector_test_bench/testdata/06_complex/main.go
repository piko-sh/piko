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

package main

import (
	// Standard library
	"net/http"

	// External dependency aliased
	oteltrace "go.opentelemetry.io/otel/trace"

	// Internal dependencies
	"testproject_complex/api"
	"testproject_complex/services"

	// External dependency (unaliased)
	"github.com/go-chi/chi/v5"
)

// Response uses types from multiple internal and external packages,
// including one with a name collision (`api.User`).
type Response struct {
	// This User is from the `api` package.
	CurrentUser api.User

	// This tests an external, aliased import.
	Span oteltrace.Span

	// This tests a type from a transitive dependency (services -> db).
	LastLogin services.LoginEvent
}

// Props shows how a component might take a complex object as a property.
type Props struct {
	// The service layer, which itself has dependencies.
	UserService services.UserService
}

// SetupRouter is a function using an external dependency.
func SetupRouter(service services.UserService) chi.Router {
	r := chi.NewRouter()
	r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
	})
	return r
}
