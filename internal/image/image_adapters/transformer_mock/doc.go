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

// Package transformer_mock is a thread-safe, in-memory test double for
// [image_domain.TransformerPort]. It supports call recording,
// configurable return values, and error simulation without requiring
// real image processing libraries.
//
// # Usage
//
//	mock := transformer_mock.NewProvider()
//	mock.SetTransformResult([]byte("output"), "image/png")
//
//	// Inject into the service under test, then assert:
//	calls := mock.GetTransformCalls()
//	assert.Equal(t, 1, len(calls))
//
// # Thread safety
//
// All methods on [Provider] are safe for concurrent use. Internal
// state is protected by a sync.RWMutex.
package transformer_mock
