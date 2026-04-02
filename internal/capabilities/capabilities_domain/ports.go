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
)

// CapabilityParams holds key-value parameters passed to a capability function.
type CapabilityParams map[string]string

// CapabilityFunc is a function type that transforms content for a specific
// capability such as compression, minification, or image processing.
type CapabilityFunc func(
	ctx context.Context,
	inputData io.Reader,
	params CapabilityParams,
) (outputData io.Reader, err error)

// CapabilityService provides methods to register and execute capabilities.
// It is used by the daemon and orchestrator to process capability requests.
type CapabilityService interface {
	// Register adds a capability function with the given name.
	//
	// Takes name (string) which identifies the capability.
	// Takes capability (CapabilityFunc) which provides the capability logic.
	//
	// Returns error when registration fails.
	Register(name string, capability CapabilityFunc) error

	// Execute runs a capability with the given input data.
	//
	// Takes capabilityName (string) which identifies the capability to run.
	// Takes inputData (io.Reader) which provides the input to process.
	// Takes params (CapabilityParams) which configures the execution.
	//
	// Returns io.Reader which provides the capability output.
	// Returns error when the capability fails or is not found.
	Execute(ctx context.Context, capabilityName string, inputData io.Reader, params CapabilityParams) (io.Reader, error)
}
