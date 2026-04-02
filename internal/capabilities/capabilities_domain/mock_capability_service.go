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
	"io"
	"sync/atomic"
)

// MockCapabilityService is a test double for CapabilityService that returns
// zero values from nil function fields and tracks call counts atomically.
type MockCapabilityService struct {
	// RegisterFunc is the function called by Register.
	RegisterFunc func(name string, capability CapabilityFunc) error

	// ExecuteFunc is the function called by Execute.
	ExecuteFunc func(ctx context.Context, capabilityName string, inputData io.Reader, params CapabilityParams) (io.Reader, error)

	// RegisterCallCount tracks how many times Register
	// was called.
	RegisterCallCount int64

	// ExecuteCallCount tracks how many times Execute was
	// called.
	ExecuteCallCount int64
}

var _ CapabilityService = (*MockCapabilityService)(nil)

// Register adds a capability function with the given name.
//
// Takes name (string) which identifies the capability by name.
// Takes capability (CapabilityFunc) which is the capability function to register.
//
// Returns error, or nil if RegisterFunc is nil.
func (m *MockCapabilityService) Register(name string, capability CapabilityFunc) error {
	atomic.AddInt64(&m.RegisterCallCount, 1)
	if m.RegisterFunc != nil {
		return m.RegisterFunc(name, capability)
	}
	return nil
}

// Execute runs a named capability with the given input data.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes capabilityName (string) which identifies the capability to execute.
// Takes inputData (io.Reader) which provides the input data stream.
// Takes params (CapabilityParams) which provides execution parameters.
//
// Returns (io.Reader, error), or (nil, nil) if ExecuteFunc is nil.
func (m *MockCapabilityService) Execute(ctx context.Context, capabilityName string, inputData io.Reader, params CapabilityParams) (io.Reader, error) {
	atomic.AddInt64(&m.ExecuteCallCount, 1)
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, capabilityName, inputData, params)
	}
	return nil, nil
}
