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

package worker_domain

import (
	"context"
	"sync"

	"piko.sh/piko/internal/worker/worker_dto"
)

// RegistryTypeErasedHandler runs a claimed job after the framework has erased its payload
// type. Registered workers are adapted to this signature.
type RegistryTypeErasedHandler func(ctx context.Context, job worker_dto.JobRecord) error

// registry is the concurrency-safe map of job kind to its type-erased worker.
type registry struct {
	// handlers maps a job kind to its registered worker.
	handlers map[string]RegistryTypeErasedHandler

	// mu guards handlers for concurrent registration and lookup.
	mu sync.RWMutex
}

// newRegistry builds an empty registry.
//
// Returns *registry which is the ready, empty registry.
func newRegistry() *registry {
	return &registry{
		handlers: make(map[string]RegistryTypeErasedHandler),
	}
}

// Lookup returns the worker registered for a kind.
//
// Takes kind (string) which is the job kind to look up.
//
// Returns RegistryTypeErasedHandler which is the registered worker, or nil when absent.
// Returns bool which is true when a worker is registered for the kind.
//
// Concurrency: acquires r.mu in read mode for the lookup.
func (r *registry) Lookup(kind string) (RegistryTypeErasedHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[kind]
	return handler, ok
}

// register stores the worker for a kind, replacing any existing registration.
//
// Takes kind (string) which is the job kind to register.
// Takes handler (RegistryTypeErasedHandler) which is the worker to run for the kind.
//
// Concurrency: acquires r.mu in write mode for the duration of the store.
func (r *registry) register(kind string, handler RegistryTypeErasedHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[kind] = handler
}
