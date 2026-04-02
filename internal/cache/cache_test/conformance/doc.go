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

// Package conformance provides a reusable conformance test suite for
// validating cache provider implementations against the cache interface
// contract.
//
// Each cache adapter can import this package and run the full suite to
// verify correct behaviour across core operations, bulk operations,
// compute operations, TTL expiry, iteration, full-text search,
// concurrent access patterns, and context cancellation.
//
// # Usage
//
//	func TestMyProvider(t *testing.T) {
//	    config := conformance.StringConfig{
//	        ProviderFactory:   myFactory,
//	        SupportsTTL:       true,
//	        SupportsIteration: true,
//	        SupportsCompute:   true,
//	    }
//	    conformance.RunStringSuite(t, config)
//	}
//
// Feature-gated sub-suites (TTL, iteration, compute, search) are
// skipped automatically when their corresponding capability flag
// is false.
package conformance
