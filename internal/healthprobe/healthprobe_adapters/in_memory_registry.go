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

package healthprobe_adapters

import (
	"sync"

	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
)

// InMemoryRegistry is a thread-safe, in-memory implementation of the Registry
// interface.
type InMemoryRegistry struct {
	// probes holds the registered health probes.
	probes []healthprobe_domain.Probe

	// mu guards access to the probes slice.
	mu sync.RWMutex
}

// Register adds a health probe to the registry.
//
// Takes probe (healthprobe_domain.Probe) which is the probe to add.
//
// Safe for concurrent use.
func (r *InMemoryRegistry) Register(probe healthprobe_domain.Probe) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.probes = append(r.probes, probe)
}

// GetAll returns all registered probes.
//
// Returns []healthprobe_domain.Probe which is a copy of the registered probes.
//
// Safe for concurrent use. Returns a copy to prevent race conditions.
func (r *InMemoryRegistry) GetAll() []healthprobe_domain.Probe {
	r.mu.RLock()
	defer r.mu.RUnlock()

	probesCopy := make([]healthprobe_domain.Probe, len(r.probes))
	copy(probesCopy, r.probes)
	return probesCopy
}

// NewInMemoryRegistry creates a new in-memory registry for health probes.
//
// Returns healthprobe_domain.Registry which is ready for registering probes.
func NewInMemoryRegistry() healthprobe_domain.Registry {
	return &InMemoryRegistry{
		probes: make([]healthprobe_domain.Probe, 0),
		mu:     sync.RWMutex{},
	}
}
