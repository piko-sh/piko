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

// Package templater_dto defines data transfer objects for the templater system.
//
// It contains the types used for communication between the HTTP layer
// and the template rendering engine, including request data, page metadata,
// caching policies, and action payloads for client-side interactions.
//
// # Request data
//
// RequestData uses object pooling to minimise allocations in high-throughput
// scenarios. Always use the builder pattern and call Release when done:
//
//	rd := templater_dto.NewRequestDataBuilder().
//	    WithContext(ctx).
//	    WithMethod("GET").
//	    WithURL(u).
//	    Build()
//	defer rd.Release()
//
// # Thread safety
//
// RequestData instances are designed to be immutable after construction.
// The builder pattern provides thread-safe construction, and getter methods
// return defensive copies where appropriate.
package templater_dto
