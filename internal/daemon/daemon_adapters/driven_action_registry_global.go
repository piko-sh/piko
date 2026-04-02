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

package daemon_adapters

import (
	"maps"
	"sync"
)

// globalActionRegistry is the global registry of action handlers.
// Actions are registered via init() functions in generated code.
var globalActionRegistry = &actionRegistry{
	entries: make(map[string]ActionHandlerEntry),
}

// actionRegistry holds registered action handlers with safe access from
// multiple goroutines.
type actionRegistry struct {
	// entries maps action names to their handler entries.
	entries map[string]ActionHandlerEntry

	// mu guards concurrent access to the registry entries.
	mu sync.RWMutex
}

// RegisterAction adds an action handler to the global registry.
// This is called by generated code in dist/actions/registry.go via init().
//
// Takes entry (ActionHandlerEntry) which describes the action handler.
//
// Safe for concurrent use by multiple goroutines.
func RegisterAction(entry ActionHandlerEntry) {
	globalActionRegistry.mu.Lock()
	defer globalActionRegistry.mu.Unlock()
	globalActionRegistry.entries[entry.Name] = entry
}

// RegisterActions adds multiple action handlers to the global registry.
// This is called by generated code in dist/actions/registry.go via init().
//
// Takes entries (map[string]ActionHandlerEntry) which maps action names
// to their handlers.
//
// Safe for concurrent use. Uses a mutex to protect the global registry.
func RegisterActions(entries map[string]ActionHandlerEntry) {
	globalActionRegistry.mu.Lock()
	defer globalActionRegistry.mu.Unlock()
	for name, entry := range entries {
		entry.Name = name
		globalActionRegistry.entries[name] = entry
	}
}

// GetGlobalActionRegistry returns a copy of all registered action handlers.
// This is called by bootstrap to get actions for the daemon.
//
// Returns map[string]ActionHandlerEntry containing all registered actions.
//
// Safe for concurrent use by multiple goroutines.
func GetGlobalActionRegistry() map[string]ActionHandlerEntry {
	globalActionRegistry.mu.RLock()
	defer globalActionRegistry.mu.RUnlock()

	result := make(map[string]ActionHandlerEntry, len(globalActionRegistry.entries))
	maps.Copy(result, globalActionRegistry.entries)
	return result
}

// ClearGlobalActionRegistry clears all registered actions.
// This is primarily used for testing.
//
// Safe for concurrent use.
func ClearGlobalActionRegistry() {
	globalActionRegistry.mu.Lock()
	defer globalActionRegistry.mu.Unlock()
	globalActionRegistry.entries = make(map[string]ActionHandlerEntry)
}
