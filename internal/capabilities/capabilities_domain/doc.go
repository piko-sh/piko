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

// Package capabilities_domain defines the port interface and service
// for pluggable content transformation capabilities.
//
// Capabilities are named functions that process streaming data, enabling
// features such as compression, minification, or image optimisation.
// The [CapabilityService] port defines the registry contract; the
// accompanying service implementation manages registration and execution.
//
// # Usage
//
//	service := capabilities_domain.NewCapabilityService(10)
//
//	// Register a capability
//	err := service.Register("minify-css", func(ctx context.Context, r io.Reader, p CapabilityParams) (io.Reader, error) {
//	    // Process and return transformed content
//	})
//
//	// Execute a registered capability
//	output, err := service.Execute(ctx, "minify-css", inputReader, params)
//
// # Thread safety
//
// All methods on [CapabilityService] are safe for concurrent use. The service
// uses a read-write mutex to allow concurrent execution whilst serialising
// registration.
package capabilities_domain
