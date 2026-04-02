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

package capabilities_domain

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// capabilityService implements the CapabilityService interface to manage and
// run registered capabilities.
type capabilityService struct {
	// capabilities maps capability names to their handler functions.
	capabilities map[string]CapabilityFunc

	// mu guards access to the capabilities map.
	mu sync.RWMutex
}

// Register adds a capability function under the given name.
//
// Takes name (string) which identifies the capability.
// Takes capability (CapabilityFunc) which provides the capability implementation.
//
// Returns error when a capability with the same name already exists.
//
// Safe for concurrent use.
func (s *capabilityService) Register(name string, capability CapabilityFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.capabilities[name]; exists {
		return fmt.Errorf("%w: %s", errCapabilityExists, name)
	}

	s.capabilities[name] = capability
	return nil
}

// Execute runs a named capability with the given input data and parameters.
//
// Takes capabilityName (string) which identifies the capability to execute.
// Takes inputData (io.Reader) which provides the input stream for processing.
// Takes params (CapabilityParams) which supplies execution parameters.
//
// Returns io.Reader which streams the capability's output.
// Returns error when the capability is not registered.
//
// Safe for concurrent use.
func (s *capabilityService) Execute(
	ctx context.Context,
	capabilityName string,
	inputData io.Reader,
	params CapabilityParams,
) (io.Reader, error) {
	s.mu.RLock()
	capability, exists := s.capabilities[capabilityName]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", errCapabilityNotFound, capabilityName)
	}

	return capability(ctx, inputData, params)
}

// NewCapabilityService creates a new instance of the CapabilityService.
//
// Takes initialCapacity (int) which is a hint to pre-allocate the underlying
// map, reducing memory re-allocations when registering a known number of
// capabilities.
//
// Returns CapabilityService which is ready to have capabilities registered.
func NewCapabilityService(initialCapacity int) CapabilityService {
	return &capabilityService{
		capabilities: make(map[string]CapabilityFunc, initialCapacity),
	}
}
