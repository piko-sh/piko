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

// Package apitest guards against unintentional API breakage using YAML golden files.
//
// It serialises exported types, functions, and interfaces into a
// human-readable YAML format that can be compared against a golden file
// baseline.
//
// # Usage
//
// Define the public API surface in a test file and call Check:
//
//	func TestAPI(t *testing.T) {
//	    surface := apitest.Surface{
//	        "MyType":     MyType{},
//	        "MyFunc":     MyFunc,
//	        "MyInterface": (*MyInterface)(nil),
//	    }
//	    apitest.Check(t, surface, "testdata/api.golden.yaml")
//	}
//
// Run with -update to create or update the golden file:
//
//	go test -update ./...
//
// # Thread safety
//
// Check is safe for concurrent use across different test functions, as each
// invocation operates on independent golden files.
package apitest
