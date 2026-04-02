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

package main

import (
	// Scenario 7: Dot import
	. "testproject_madness/pkg/dot"
	// Scenario 8: Blank import
	_ "testproject_madness/pkg/blank"
	// Scenario 6: Alias to avoid stdlib collision
	myhttp "testproject_madness/pkg/http"
	// Standard library for comparison
	"net/http"
	// Scenario 4 & 5: Alias one `api` package
	apiv1 "testproject_madness/pkg/api"
	// Use the default name for the other `api` package
	"testproject_madness/pkg/api/v2"
	// Scenario 1 & 2: Alias our local `helpers` package
	localhelpers "testproject_madness/pkg/utils"
	// Use the default name for the external `helpers` package
	"testproject_madness/third_party/github.com/external/helpers"
)

// The Response struct uses types from every chaotic scenario.
type Response struct {
	// Test Scenario 7: Dot imported type is used without a qualifier.
	DotInfo DotImported

	// Test Scenario 4 & 5: Distinguishing between two packages named `api`.
	APIUser    apiv1.User  // Should resolve to `pkg/api`
	APIProduct api.Product // Should resolve to `pkg/api/v2`

	// Test Scenario 1, 2, 9 & 10: Distinguishing between two packages named `helpers`.
	Local    localhelpers.UtilHelper      // Should resolve to `pkg/utils`
	External helpers.ExternalHelper       // Should resolve to `vendor/.../helpers`
	Boxed    localhelpers.Box[apiv1.User] // Test generics from an aliased, mismatched package
	Embedded                              // Anonymous embedding to test method promotion

	StdlibClient http.Client
}

// Embedded struct for Scenario 10
type Embedded struct {
	// Embed a type from a package that was aliased to avoid collision
	myhttp.Client
}
